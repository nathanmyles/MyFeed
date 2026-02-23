import { app, BrowserWindow, ipcMain } from 'electron'
import { spawn, ChildProcess } from 'node:child_process'
import { fileURLToPath } from 'node:url'
import path from 'node:path'
import fs from 'node:fs'
import os from 'node:os'

const __dirname = path.dirname(fileURLToPath(import.meta.url))

process.env.APP_ROOT = path.join(__dirname, '..')

export const VITE_DEV_SERVER_URL = process.env['VITE_DEV_SERVER_URL']
export const MAIN_DIST = path.join(process.env.APP_ROOT, 'dist-electron')
export const RENDERER_DIST = path.join(process.env.APP_ROOT, 'dist')

process.env.VITE_PUBLIC = VITE_DEV_SERVER_URL ? path.join(process.env.APP_ROOT, 'public') : RENDERER_DIST

let win: BrowserWindow | null
let daemonProcess: ChildProcess | null = null

function getDataDir(): string {
  const homeDir = os.homedir()
  return path.join(homeDir, '.myfeed')
}

function getDaemonPath(): string {
  if (VITE_DEV_SERVER_URL) {
    return path.join(process.env.APP_ROOT, 'resources', 'daemon', 'myfeed-daemon')
  }
  const platform = process.platform
  const ext = platform === 'win32' ? '.exe' : ''
  return path.join(process.resourcesPath, 'daemon', `myfeed-daemon${ext}`)
}

function startDaemon() {
  const daemonPath = getDaemonPath()
  const dataDir = getDataDir()
  
  if (!fs.existsSync(daemonPath)) {
    console.error('Daemon binary not found at:', daemonPath)
    return
  }

  if (!fs.existsSync(dataDir)) {
    fs.mkdirSync(dataDir, { recursive: true })
  }

  console.log('Starting daemon from:', daemonPath)
  console.log('Data directory:', dataDir)

  daemonProcess = spawn(daemonPath, ['-data', dataDir], {
    stdio: ['ignore', 'pipe', 'pipe']
  })

  daemonProcess.stdout?.on('data', (data) => {
    console.log(`[daemon] ${data}`)
  })

  daemonProcess.stderr?.on('data', (data) => {
    console.error(`[daemon error] ${data}`)
  })

  daemonProcess.on('close', (code) => {
    console.log(`Daemon process exited with code ${code}`)
    daemonProcess = null
  })

  daemonProcess.on('error', (err) => {
    console.error('Failed to start daemon:', err)
  })
}

function stopDaemon() {
  if (daemonProcess) {
    console.log('Stopping daemon...')
    daemonProcess.kill()
    daemonProcess = null
  }
}

function createWindow() {
  win = new BrowserWindow({
    width: 1200,
    height: 800,
    icon: path.join(process.env.VITE_PUBLIC, 'electron-vite.svg'),
    webPreferences: {
      preload: path.join(__dirname, 'preload.mjs'),
    },
  })

  win.webContents.on('did-finish-load', () => {
    win?.webContents.send('main-process-message', (new Date).toLocaleString())
  })

  if (VITE_DEV_SERVER_URL) {
    win.loadURL(VITE_DEV_SERVER_URL)
  } else {
    win.loadFile(path.join(RENDERER_DIST, 'index.html'))
  }
}

app.on('window-all-closed', () => {
  stopDaemon()
  if (process.platform !== 'darwin') {
    app.quit()
    win = null
  }
})

app.on('activate', () => {
  if (BrowserWindow.getAllWindows().length === 0) {
    createWindow()
  }
})

app.on('before-quit', () => {
  stopDaemon()
})

app.whenReady().then(() => {
  ipcMain.handle('get-home-dir', () => os.homedir())

ipcMain.handle('read-port-file', async () => {
  const portFile = path.join(getDataDir(), 'daemon.port')
  try {
    const port = fs.readFileSync(portFile, 'utf-8').trim()
    return parseInt(port, 10)
  } catch {
    return null
  }
})
  startDaemon()
  createWindow()
})
