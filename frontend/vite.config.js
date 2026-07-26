import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// https://vite.dev/config/
export default defineConfig({
  plugins: [react()],
  server: {
    proxy: {
      '/login': 'http://localhost:8080',
      '/users': 'http://localhost:8080',
      '/me': 'http://localhost:8080',
      '/verify': 'http://localhost:8080',
      '/logout': 'http://localhost:8080',
      '/refresh': 'http://localhost:8080',
      '/chats': 'http://localhost:8080',
      '/messages': 'http://localhost:8080',
      '/sessions': 'http://localhost:8080',
    },
  },
})
