# --- Build stage: compile the React app to static assets ------------------
FROM node:24-alpine AS build
WORKDIR /app

# Install deps first for better layer caching.
COPY package.json ./
RUN npm install

COPY . .
RUN npm run build

# --- Serve stage: tiny static server (no nginx) ---------------------------
# `serve -s` serves the SPA with a history-API fallback to index.html, so deep
# links / refreshes work. Traefik routes traffic here; everything under /api is
# routed to the backend instead.
FROM node:24-alpine AS serve
WORKDIR /app
RUN npm install -g serve
COPY --from=build /app/dist ./dist
EXPOSE 80
CMD ["serve", "-s", "dist", "-l", "80"]
