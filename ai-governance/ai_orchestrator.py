import os
import datetime
import json
from dotenv import load_dotenv
from crewai import Agent, Task, Crew, Process
from crewai.tools import tool
from git import Repo, exc

load_dotenv()

# --- CONFIGURATION & PATHS ---
AUDIT_LOG_PATH = "ai-governance/audit.log"
GUIDELINES_PATH = "ai-governance/architecture_guidelines.txt"

# RBAC ZONES (Plan 2)
ALLOWED_WRITE_ZONES = ["web/client/", "web/server/", "mobile/app/", "README.md", "ai-governance/"]
FORBIDDEN_WRITE_ZONES = ["web/secrets/", "ai-governance/audit.log", ".git/", ".env", ".github/"]
SENSITIVE_READ_ZONES = ["web/secrets/"]

def audit_log(agent_name, action, details):
    """Plan 5 (Evidence) - Immutable local logging."""
    timestamp = datetime.datetime.now().strftime("%Y-%m-%d %H:%M:%S")
    # Convert details to string if it's not already
    if not isinstance(details, str):
        try:
            details = json.dumps(details)
        except:
            details = str(details)
            
    entry = f"[{timestamp}] AGENT:{agent_name} | ACTION:{action} | DETAILS:{details}\n"
    os.makedirs(os.path.dirname(AUDIT_LOG_PATH), exist_ok=True)
    with open(AUDIT_LOG_PATH, "a") as f:
        f.write(entry)
    print(entry.strip())

def log_step(step):
    """Callback to capture the 'tough process' and tool invocations of the agent."""
    audit_log("AgentReasoning", "STEP_PROCESS", {
        "thought": getattr(step, 'thought', 'N/A'),
        "tool": getattr(step, 'tool', 'N/A'),
        "tool_input": getattr(step, 'tool_input', 'N/A'),
        "observation": getattr(step, 'observation', 'N/A')
    })

def log_task_output(task_output):
    """Callback to capture the final answer of a task."""
    audit_log("AgentFinalOutput", "TASK_COMPLETED", {
        "task": task_output.description,
        "raw_output": task_output.raw,
        "summary": task_output.summary
    })

# --- PLAN 2: CONTROL PLANE (CAPABILITY GATEWAY / MCP) ---
class CapabilityGateway:
    @staticmethod
    def _get_repo():
        try:
            return Repo(os.getcwd(), search_parent_directories=True)
        except exc.InvalidGitRepositoryError:
            return None

    @staticmethod
    def _validate_access(path: str, mode: str = "read"):
        """Enforces RBAC and Security Engine rules (Plan 2)."""
        # 1. Check for sensitive reads
        if mode == "read":
            if any(path.startswith(z) for z in SENSITIVE_READ_ZONES):
                audit_log("SecurityEngine", "SENSITIVE_READ_WARNING", path)
                # We allow reading for context, but it's heavily audited
                return True, ""
        
        # 2. Check for unauthorized writes (ZONE_RED / FORBIDDEN)
        if mode == "write":
            if any(path.startswith(z) for z in FORBIDDEN_WRITE_ZONES):
                audit_log("SecurityEngine", "RBAC_VIOLATION_BLOCK", f"Write denied to {path} (ZONE_RED)")
                return False, f"Permission Denied: {path} is in a protected ZONE_RED."
            
            if not any(path.startswith(z) for z in ALLOWED_WRITE_ZONES):
                audit_log("SecurityEngine", "RBAC_OUT_OF_BOUNDS", f"Write denied to {path}")
                return False, f"Permission Denied: {path} is outside allowed developer zones."
        
        return True, ""

    @tool("read_file")
    def read_file(path: str):
        """Read a file to gain project context (Plan 1)."""
        allowed, msg = CapabilityGateway._validate_access(path, mode="read")
        if not allowed: return msg

        audit_log("Gateway", "READ_FILE", path)
        try:
            repo = CapabilityGateway._get_repo()
            full_path = os.path.join(repo.working_tree_dir, path) if repo else path
            with open(full_path, 'r') as f:
                return f.read()
        except Exception as e:
            return f"Error: {str(e)}"

    @tool("write_file")
    def write_file(path: str, content: str):
        """Propose a code change. Strictly restricted by RBAC (Plan 2)."""
        allowed, msg = CapabilityGateway._validate_access(path, mode="write")
        if not allowed: return msg
        
        audit_log("Gateway", "WRITE_FILE", path)
        try:
            repo = CapabilityGateway._get_repo()
            full_path = os.path.join(repo.working_tree_dir, path) if repo else path
            os.makedirs(os.path.dirname(full_path), exist_ok=True)
            with open(full_path, 'w') as f:
                f.write(content)
            return f"Success: Modified {path}"
        except Exception as e:
            return f"Error: {str(e)}"

    @tool("list_directory")
    def list_directory(path: str = "."):
        """List files and directories in a given path to explore project structure."""
        audit_log("Gateway", "LIST_DIR", path)
        try:
            repo = CapabilityGateway._get_repo()
            full_path = os.path.join(repo.working_tree_dir, path) if repo else path
            if os.path.isdir(full_path):
                return os.listdir(full_path)
            else:
                return f"Error: {path} is not a directory."
        except Exception as e:
            return f"Error: {str(e)}"

    @tool("check_server_compilation")
    def check_server_compilation():
        """Check if the Go backend (web/server) compiles correctly."""
        audit_log("Gateway", "CHECK_SERVER_BUILD", "Compiling Go backend")
        import subprocess
        try:
            repo = CapabilityGateway._get_repo()
            root = repo.working_tree_dir if repo else os.getcwd()
            server_path = os.path.join(root, "web/server")
            
            # Using 'go build ./...' to check for compilation errors without producing a binary
            result = subprocess.run(
                ["go", "build", "./..."], 
                cwd=server_path, 
                capture_output=True, 
                text=True, 
                timeout=60
            )
            
            if result.returncode == 0:
                return "Success: Server compiles perfectly."
            else:
                return f"Compilation Failed:\nSTDOUT: {result.stdout}\nSTDERR: {result.stderr}"
        except Exception as e:
            return f"Error: {str(e)}"

    @tool("check_client_compilation")
    def check_client_compilation():
        """Check if the React frontend (web/client) compiles correctly."""
        audit_log("Gateway", "CHECK_CLIENT_BUILD", "Compiling React frontend")
        import subprocess
        try:
            repo = CapabilityGateway._get_repo()
            root = repo.working_tree_dir if repo else os.getcwd()
            client_path = os.path.join(root, "web/client")
            
            # Using 'yarn build' as requested by the user
            result = subprocess.run(
                ["yarn", "build"], 
                cwd=client_path, 
                capture_output=True, 
                text=True, 
                timeout=120
            )
            
            if result.returncode == 0:
                return "Success: Client compiles perfectly."
            else:
                return f"Compilation Failed:\nSTDOUT: {result.stdout}\nSTDERR: {result.stderr}"
        except Exception as e:
            return f"Error: {str(e)}"

    @tool("get_git_diff")
    def get_git_diff():
        """Get the current staged/unstaged changes in the repository to verify modifications."""
        audit_log("Gateway", "GIT_DIFF", "Current changes")
        try:
            repo = CapabilityGateway._get_repo()
            if not repo: return "Error: Git repo not found."
            return repo.git.diff()
        except Exception as e:
            return f"Error: {str(e)}"

    @tool("quarantine_branch")
    def create_quarantine_branch(branch_name: str):
        """Create an isolated sandbox branch for AI changes (Plan 3)."""
        audit_log("Gateway", "QUARANTINE_BRANCH", branch_name)
        try:
            repo = CapabilityGateway._get_repo()
            if not repo: return "Error: Git repo not found."
            
            # Ensure naming convention for Plan 3
            if not branch_name.startswith("quarantine/"):
                branch_name = f"quarantine/{branch_name}"
            
            try:
                # Create and checkout
                repo.git.checkout('-b', branch_name)
                return f"Success: Switched to sandbox branch {branch_name}"
            except exc.GitCommandError:
                # Fallback: Just checkout if it exists
                repo.git.checkout(branch_name)
                return f"Success: Using existing branch {branch_name}"
        except Exception as e:
            return f"Error: {str(e)}"

# --- PLAN 1: KNOWLEDGE & RETRIEVAL (RAG) ---
def load_rag_context():
    """Loads the hardcoded security blueprint and guidelines (Plan 1)."""
    try:
        with open(GUIDELINES_PATH, "r") as f:
            return f"\n[ARCHITECTURAL BLUEPRINT & GUIDELINES]\n{f.read()}\n"
    except FileNotFoundError:
        return "[SECURE RAG] Critical Error: Architecture guidelines missing."

# --- AGENT SWARM (PLAN 2) ---
llm_model = os.environ.get("AGENT_MODEL", "gpt-4o-mini")

developer_agent = Agent(
    role='Security-Conscious Developer',
    goal='Implement tasks within quarantine zones while following RBAC rules.',
    backstory="""You are an expert developer. You are explicitly aware that git branch 
    names have a length restriction (max 50 chars) and should be safe (no spaces). 
    You always use the exact branch name provided in your task.
    You use 'check_server_compilation' and 'check_client_compilation' to verify your changes.""",
    tools=[CapabilityGateway.read_file, CapabilityGateway.write_file, 
           CapabilityGateway.create_quarantine_branch, CapabilityGateway.list_directory,
           CapabilityGateway.check_server_compilation, CapabilityGateway.check_client_compilation, 
           CapabilityGateway.get_git_diff],
    llm=llm_model,
    verbose=True
)

# --- PLAN 1: KNOWLEDGE & RETRIEVAL (PROTECTED RAG MODULE) ---
class ProjectRAG:
    """Provides HIGH-LEVEL project context. For file-specific logic, use search tools."""
    RAG_DIR = "ai-governance/rag/"

    @staticmethod
    @tool("query_project_knowledge")
    def query_knowledge(topic: str):
        """Query the protected knowledge base for HIGH-LEVEL project metadata, security schemas, 
        topology, or architectural guidelines. Topics: 'topology', 'security', 'mtls', 'rbac', 'standards'."""
        audit_log("RAG", "QUERY", topic)
        try:
            results = []
            topic = topic.lower()
            
            # Map of topics to their markdown files
            files = {
                "topology": "topology.md",
                "security": "security_logic.md",
                "components": "component_summaries.md",
                "flows": "data_flows.md",
                "standards": "development_standards.md"
            }
            
            # If specific topic requested
            if topic in files:
                path = os.path.join(ProjectRAG.RAG_DIR, files[topic])
                with open(path, 'r') as f:
                    return f.read()

            # Otherwise search through all
            for filename in files.values():
                path = os.path.join(ProjectRAG.RAG_DIR, filename)
                if os.path.exists(path):
                    with open(path, 'r') as f:
                        content = f.read()
                        if topic in content.lower():
                            results.append(f"--- FROM {filename} (HIGH-LEVEL) ---\n{content}")
            
            return "\n\n".join(results) if results else "No high-level knowledge found for topic."
            
        except Exception as e:
            return f"Error querying RAG: {str(e)}"

    @staticmethod
    @tool("search_project_files")
    def search_files(query: str):
        """Search for files by NAME/PATH. Supports multiple keywords (e.g., 'admin label')."""
        audit_log("RAG", "SEARCH_PATH", query)
        repo = CapabilityGateway._get_repo()
        root = repo.working_tree_dir if repo else os.getcwd()
        
        keywords = query.lower().split()
        results = []
        for dir_path, _, filenames in os.walk(root):
            if any(p in dir_path for p in [".git", "node_modules", ".venv", "rag"]):
                continue
            for f in filenames:
                rel_path = os.path.relpath(os.path.join(dir_path, f), root).lower()
                if all(kw in rel_path for kw in keywords):
                    results.append(os.path.relpath(os.path.join(dir_path, f), root))
        
        return results[:20]

    @staticmethod
    @tool("search_file_contents")
    def search_contents(keyword: str):
        """Search for STRINGs or WORDs inside ALLOWED project files. Supports multiple keywords."""
        audit_log("RAG", "SEARCH_CONTENT", keyword)
        repo = CapabilityGateway._get_repo()
        root = repo.working_tree_dir if repo else os.getcwd()
        
        keywords = keyword.lower().split()
        results = []
        for dir_path, _, filenames in os.walk(root):
            # Exclude sensitive and non-project directories
            if any(p in dir_path for p in [".git", "node_modules", ".venv", "rag", "web/secrets"]):
                continue
                
            # Only search in allowed developer zones for logic/words
            rel_dir = os.path.relpath(dir_path, root)
            if rel_dir != "." and not any(rel_dir.startswith(z) for z in ALLOWED_WRITE_ZONES):
                continue

            for f in filenames:
                # Target relevant source and config files
                if not f.endswith(('.go', '.tsx', '.ts', '.css', '.html', '.kts', '.md', '.json', '.yaml', '.yml', '.sql')):
                    continue
                path = os.path.join(dir_path, f)
                try:
                    with open(path, 'r', encoding='utf-8') as file:
                        content = file.read().lower()
                        if all(kw in content for kw in keywords):
                            results.append(os.path.relpath(path, root))
                except:
                    continue
        
        return results[:20]

# --- AGENT SWARM (SIMPLIFIED PLAN 2) ---
llm_model = os.environ.get("AGENT_MODEL", "gpt-4o-mini")

lead_engineer = Agent(
    role='Lead AI Engineer',
    goal='Accurately implement requirements using RAG to find files and follow standards.',
    backstory="""You are a senior engineer who doesn't guess. 
    1. You use 'query_project_knowledge' to understand architecture.
    2. You use 'search_file_contents' to find exactly where UI text or logic is defined.
    3. You use 'search_project_files' to locate the specific component files.
    4. You use compilation check tools to validate your work.
    You create the quarantine branch, apply fixes, and ensure they work perfectly.""",
    tools=[CapabilityGateway.read_file, CapabilityGateway.write_file, 
           CapabilityGateway.create_quarantine_branch, CapabilityGateway.list_directory,
           CapabilityGateway.check_server_compilation, CapabilityGateway.check_client_compilation, 
           CapabilityGateway.get_git_diff,
           ProjectRAG.search_files, ProjectRAG.query_knowledge, ProjectRAG.search_contents],
    llm=llm_model,
    verbose=True,
    max_iter=15
)

auditor = Agent(
    role='Security & Quality Auditor',
    goal='Verify implementation against the official RAG schemas and guidelines.',
    backstory=f"""You are the final gatekeeper. You review code for logic errors, 
    quality, and RBAC compliance using the official knowledge base.
    
    IMPORTANT: Use 'get_git_diff' to see exactly what changed.
    Use compilation check tools to verify the build is not broken.
    Use 'search_file_contents' to verify your findings.
    You MUST delegate back to the Lead Engineer if any issues exist. 
    Finish ONLY with a 'FINAL APPROVAL & SECURITY PASSED' message.""",
    tools=[CapabilityGateway.read_file, CapabilityGateway.get_git_diff, 
           CapabilityGateway.list_directory, CapabilityGateway.check_server_compilation, 
           CapabilityGateway.check_client_compilation,
           ProjectRAG.search_files, ProjectRAG.query_knowledge, ProjectRAG.search_contents],
    llm=llm_model,
    verbose=True,
    allow_delegation=True,
    max_iter=10
)

import re

def slugify(text):
    """Explicitly converts text to a safe git branch slug (max 50 chars)."""
    text = text.lower()
    text = re.sub(r'[^a-z0-9]+', '-', text)
    return text.strip('-')[:50]

def run_secure_orchestrator(user_task):
    audit_log("Orchestrator", "TASK_INIT", {"prompt": user_task})
    feature_slug = slugify(user_task)
    branch_name = f"quarantine/{feature_slug}"
    
    # 1. Implementation Task (Lead Engineer)
    t1 = Task(
        description=f"""Task: {user_task}. 
        1. Create branch '{branch_name}'. 
        2. Use 'search_project_files' and 'list_directory' to find where the relevant code resides.
        3. Implement the requirement perfectly.
        4. Use 'check_server_compilation' and 'check_client_compilation' to verify your work.
        5. Do NOT guess paths.""",
        agent=lead_engineer,
        callback=log_task_output,
        expected_output=f"A definitive report of changes in {branch_name}."
    )

    # 2. Verification Task (Auditor)
    t2 = Task(
        description=f"""Review the work in '{branch_name}' for task: '{user_task}'.
        Ensure quality, logic, and RBAC security. 
        Use 'get_git_diff' to see exactly what changed.
        Use compilation check tools to verify the build.
        Delegate fixes if needed. Finalize only when perfect.""",
        agent=auditor,
        callback=log_task_output,
        context=[t1], # Explicitly aware of what was done in T1
        expected_output="A final statement: 'FINAL APPROVAL & SECURITY PASSED'."
    )

    crew = Crew(
        agents=[lead_engineer, auditor],
        tasks=[t1, t2],
        process=Process.sequential,
        step_callback=log_step
    )

    # EXECUTION
    result = crew.kickoff()
    
    audit_log("Orchestrator", "PROPOSAL_READY", f"Changes staged in local {branch_name} branch.")
    print("\n" + "="*50)
    print("PLAN 2: HUMAN SUPERVISION REQUIRED")
    print(f"AI has proposed changes in branch: {branch_name}")
    print("Please review locally and merge manually if approved.")
    print("="*50)

    return result

if __name__ == "__main__":
    import sys
    if len(sys.argv) > 1:
        run_secure_orchestrator(sys.argv[1])
    else:
        print("Usage: python3 ai_orchestrator.py \"<task description>\"")
