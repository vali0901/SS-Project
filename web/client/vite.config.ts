import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'
import * as fs from 'node:fs'
import * as https from 'node:https'

const secret = (name: string) => `/run/secrets/${name}`
const secretExists = (name: string) => fs.existsSync(secret(name))

const tlsSecretsAvailable =
  secretExists('web.crt') && secretExists('web.key') && secretExists('ca.crt')

const httpsOptions = tlsSecretsAvailable
  ? {
      cert: fs.readFileSync(secret('web.crt')),
      key: fs.readFileSync(secret('web.key')),
    }
  : undefined

const backendProxyAgent = tlsSecretsAvailable
  ? new https.Agent({
      ca: fs.readFileSync(secret('ca.crt')),
    })
  : undefined

// https://vite.dev/config/
export default defineConfig({
  plugins: [react(), tailwindcss()],
  cacheDir: '.vite',
  server: {
    https: httpsOptions,
    proxy: {
      '/api': {
        target: 'https://go-api:8443',
        changeOrigin: true,
        secure: true,
        agent: backendProxyAgent,
        rewrite: (path) => path.replace(/^\/api/, ''),
      },
    },
  },
})
