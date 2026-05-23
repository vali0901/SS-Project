FROM node:22-alpine

WORKDIR /app

# Use the node user for better security and ownership
RUN mkdir -p /app && chown -R node:node /app
USER node

# Install dependencies first (better caching)
COPY --chown=node:node package.json yarn.lock ./
RUN yarn install --frozen-lockfile

# Copy the rest of the application
COPY --chown=node:node . .

# Set environment variables
ENV CHOKIDAR_USEPOLLING=true

# Expose the Vite port
EXPOSE 5173

# Default command
CMD ["yarn", "dev:poll", "--host", "0.0.0.0", "--port", "5173"]
