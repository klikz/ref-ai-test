import beepOkUrl from "@/assets/sounds/beep_ok.mp3"
import alertUrl from "@/assets/sounds/alert.mp3"

let okAudio: HTMLAudioElement | null = null
let alertAudio: HTMLAudioElement | null = null
let unlocked = false

function ensureAudio() {
  if (!okAudio) {
    okAudio = new Audio(beepOkUrl)
  }
  if (!alertAudio) {
    alertAudio = new Audio(alertUrl)
  }
}

function unlockAudio() {
  if (unlocked) {
    return
  }
  unlocked = true
  ensureAudio()
  for (const audio of [okAudio, alertAudio]) {
    if (!audio) {
      continue
    }
    audio.muted = true
    void audio
      .play()
      .then(() => {
        audio.pause()
        audio.currentTime = 0
        audio.muted = false
      })
      .catch(() => {
        audio.muted = false
      })
  }
}

function bindUnlockListeners() {
  if (typeof window === "undefined") {
    return
  }
  const events = ["pointerdown", "keydown", "touchstart"] as const
  const onInteract = () => {
    unlockAudio()
    for (const event of events) {
      window.removeEventListener(event, onInteract)
    }
  }
  for (const event of events) {
    window.addEventListener(event, onInteract, { once: true, passive: true })
  }
}

bindUnlockListeners()

function play(audio: HTMLAudioElement) {
  unlockAudio()
  audio.muted = false
  audio.currentTime = 0
  void audio.play().catch(() => {})
}

export function playScanOk() {
  ensureAudio()
  if (okAudio) {
    play(okAudio)
  }
}

export function playScanError() {
  ensureAudio()
  if (alertAudio) {
    play(alertAudio)
  }
}
