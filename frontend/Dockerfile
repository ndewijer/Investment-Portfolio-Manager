# Build stage
FROM node:24-alpine AS build

# Set working directory
WORKDIR /app

# Copy package files
COPY package*.json ./

# Install dependencies
RUN npm ci

# Copy the rest of the application
COPY . .

# Build the application
# No build args needed - API URL is determined at runtime via nginx proxy
RUN npm run build

# Production stage
FROM nginx:alpine

# Copy built assets from build stage
COPY --from=build /app/build /usr/share/nginx/html

# Copy nginx configuration as template
# BACKEND_HOST is set via environment variable at runtime
COPY nginx.conf /etc/nginx/templates/default.conf.template

CMD ["nginx", "-g", "daemon off;"]
