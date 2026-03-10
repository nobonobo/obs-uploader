import { defineConfig } from "vite";
import { svelte } from "@sveltejs/vite-plugin-svelte";
import tailwindcss from "@tailwindcss/vite";
import wails from "@wailsio/runtime/plugins/vite";

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [
    tailwindcss(), 
    svelte(), 
    wails("./bindings"),
    {
      name: 'wails-custom-js-mock',
      configureServer(server) {
        server.middlewares.use((req, res, next) => {
          if (req.url === '/wails/custom.js') {
            res.setHeader('Content-Type', 'application/javascript');
            res.end('');
            return;
          }
          next();
        });
      }
    }
  ],
});
