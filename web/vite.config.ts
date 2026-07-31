import { defineConfig } from "vite";
import vue from "@vitejs/plugin-vue";
import tailwindcss from "@tailwindcss/vite";

export default defineConfig({
  plugins: [vue(), tailwindcss()],
  server: {
    proxy: {
      "/api": "http://localhost:8080",
      "/alpine": "http://localhost:8080",
      "/apt": "http://localhost:8080",
      "/cargo": "http://localhost:8080",
      "/conda": "http://localhost:8080",
      "/cran": "http://localhost:8080",
      "/docker": "http://localhost:8080",
      "/gh": "http://localhost:8080",
      "/ghapi": "http://localhost:8080",
      "/ghcr": "http://localhost:8080",
      "/gl": "http://localhost:8080",
      "/golang": "http://localhost:8080",
      "/homebrew": "http://localhost:8080",
      "/hf": "http://localhost:8080",
      "/mcr": "http://localhost:8080",
      "/npm": "http://localhost:8080",
      "/nuget": "http://localhost:8080",
      "/pypi": "http://localhost:8080",
      "/quay": "http://localhost:8080",
      "/rubygems": "http://localhost:8080",
      "/v2": "http://localhost:8080",
    },
  },
});
