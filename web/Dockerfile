# Build the Vue.js application
FROM oven/bun:1-alpine AS build
WORKDIR /app

ARG VITE_WS_URL
ENV VITE_WS_URL=$VITE_WS_URL

COPY package.json package.json ./
COPY bun.lock bun.lock
RUN bun install --frozen-lockfile

COPY . .
RUN bun run build

# Final Nginx container
FROM nginx:alpine
COPY nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=build /app/dist /usr/share/nginx/html

EXPOSE 8080
