# --- Build stage: compile the React app to static assets ------------------
FROM node:24-alpine AS build
WORKDIR /app

# Install deps first for better layer caching.
COPY package.json ./
RUN npm install

COPY . .
RUN npm run build

# --- Serve stage: nginx serving the static build --------------------------
# nginx.conf adds a history-API fallback to index.html so deep links /
# refreshes work. Traefik routes traffic here; everything under /api is routed
# to the backend instead.
FROM nginx:alpine AS serve
COPY nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=build /app/dist /usr/share/nginx/html
EXPOSE 80
CMD ["nginx", "-g", "daemon off;"]
