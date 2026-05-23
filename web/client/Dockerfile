FROM node:22-alpine

WORKDIR /app

# Install dependencies first (better caching)
COPY package.json yarn.lock ./
RUN yarn install --frozen-lockfile

# Copy the rest of the application
COPY . .

# Make node_modules writable by any user (for anonymous volume with custom UID)
RUN chmod -R 777 /app/node_modules

# Set environment variables
ENV CHOKIDAR_USEPOLLING=true

# Expose the Vite port
EXPOSE 5173

# Default command
CMD ["yarn", "dev:poll", "--host", "0.0.0.0", "--port", "5173"]
