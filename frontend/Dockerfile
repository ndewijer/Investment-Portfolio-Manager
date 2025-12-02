# Build stage
FROM node:24-alpine AS build

# Set working directory
WORKDIR /app

# Add build arguments
ARG DOMAIN=localhost \
    USE_HTTPS=false
ENV DOMAIN=$DOMAIN \
    USE_HTTPS=$USE_HTTPS

# Copy package files
COPY package*.json ./

# Install dependencies
RUN npm ci

# Copy the rest of the application
COPY . .

# Build the application
RUN npm run build

# Production stage
FROM nginx:alpine

# Re-declare the build argument and env variable in this stage
ARG DOMAIN=localhost \
    USE_HTTPS=false
ENV DOMAIN=$DOMAIN \
    USE_HTTPS=$USE_HTTPS

# Copy built assets from build stage
COPY --from=build /app/build /usr/share/nginx/html

# Copy nginx configuration
COPY nginx.conf /etc/nginx/conf.d/nginx.conf

RUN envsubst '${DOMAIN}' < /etc/nginx/conf.d/nginx.conf > /etc/nginx/conf.d/default.conf && \
    rm /etc/nginx/conf.d/nginx.conf

CMD ["nginx", "-g", "daemon off;"]
