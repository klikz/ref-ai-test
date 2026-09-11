import { useCallback, useEffect, useRef, useState } from "react"
import { Camera, RefreshCw, VideoOff } from "lucide-react"
import { PageContainer } from "@/components/layout/page-container"
import { Panel } from "@/components/layout/panel"
import { Button } from "@/components/ui/button"
import { Label } from "@/components/ui/label"
import { cn } from "@/lib/utils"

type CameraDevice = {
  deviceId: string
  label: string
}

export default function CameraTestPage() {
  const videoRef = useRef<HTMLVideoElement>(null)
  const streamRef = useRef<MediaStream | null>(null)
  const [devices, setDevices] = useState<CameraDevice[]>([])
  const [selectedDeviceId, setSelectedDeviceId] = useState("")
  const [online, setOnline] = useState(false)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState("")

  const stopStream = useCallback(() => {
    if (streamRef.current) {
      for (const track of streamRef.current.getTracks()) {
        track.stop()
      }
      streamRef.current = null
    }
    if (videoRef.current) {
      videoRef.current.srcObject = null
    }
    setOnline(false)
  }, [])

  const loadDevices = useCallback(async () => {
    if (!navigator.mediaDevices?.enumerateDevices) {
      return []
    }
    const allDevices = await navigator.mediaDevices.enumerateDevices()
    return allDevices
      .filter((device) => device.kind === "videoinput")
      .map((device, index) => ({
        deviceId: device.deviceId,
        label: device.label || `Kamera ${index + 1}`,
      }))
  }, [])

  const startCamera = useCallback(
    async (deviceId?: string) => {
      if (!navigator.mediaDevices?.getUserMedia) {
        setError("Brauzer kamera API ni qo'llab-quvvatlamaydi")
        setLoading(false)
        setOnline(false)
        return
      }

      setLoading(true)
      setError("")
      stopStream()

      try {
        const constraints: MediaStreamConstraints = {
          video: deviceId
            ? { deviceId: { exact: deviceId } }
            : { facingMode: "environment" },
          audio: false,
        }
        const stream = await navigator.mediaDevices.getUserMedia(constraints)
        streamRef.current = stream

        const video = videoRef.current
        if (video) {
          video.srcObject = stream
          await video.play()
        }

        const cameraDevices = await loadDevices()
        setDevices(cameraDevices)
        const activeTrack = stream.getVideoTracks()[0]
        const activeDeviceId = activeTrack?.getSettings().deviceId ?? deviceId ?? ""
        if (activeDeviceId) {
          setSelectedDeviceId(activeDeviceId)
        } else if (cameraDevices.length > 0) {
          setSelectedDeviceId(cameraDevices[0].deviceId)
        }

        setOnline(true)
      } catch (err) {
        setOnline(false)
        setError(err instanceof Error ? err.message : "Kameraga ulanib bo'lmadi")
      } finally {
        setLoading(false)
      }
    },
    [loadDevices, stopStream],
  )

  useEffect(() => {
    void startCamera()
    return () => {
      stopStream()
    }
  }, [startCamera, stopStream])

  return (
    <PageContainer
      title="Camera Test"
      description="Jonli kamera ko'rinishi"
      fullWidth
      actions={
        <Button
          type="button"
          variant="outline"
          className="gap-2 rounded-xl"
          disabled={loading}
          onClick={() => void startCamera(selectedDeviceId || undefined)}
        >
          <RefreshCw className={cn("size-4", loading && "animate-spin")} />
          Qayta ulanish
        </Button>
      }
    >
      <div className="grid gap-4 xl:grid-cols-[minmax(0,1fr)_320px]">
        <Panel title="Jonli ko'rinish" noPadding>
          <div className="relative aspect-video w-full overflow-hidden rounded-b-xl bg-black">
            <video
              ref={videoRef}
              autoPlay
              playsInline
              muted
              className={cn("h-full w-full object-contain", !online && "opacity-0")}
            />
            {!online && !loading ? (
              <div className="absolute inset-0 flex flex-col items-center justify-center gap-3 text-muted-foreground">
                <VideoOff className="size-12 opacity-60" />
                <p className="text-sm">Kamera offline</p>
              </div>
            ) : null}
            {loading ? (
              <div className="absolute inset-0 flex items-center justify-center bg-black/60 text-sm text-white">
                Kamera ulanmoqda...
              </div>
            ) : null}
            <div
              className={cn(
                "absolute top-4 left-4 inline-flex items-center gap-2 rounded-full px-3 py-1 text-xs font-semibold",
                online ? "bg-emerald-600 text-white" : "bg-red-600 text-white",
              )}
            >
              <span className={cn("size-2 rounded-full", online ? "bg-white animate-pulse" : "bg-white/80")} />
              {online ? "Online" : "Offline"}
            </div>
          </div>
        </Panel>

        <Panel title="Holat">
          <div className="space-y-4">
            <div className="flex items-start gap-3 rounded-xl border border-border/60 bg-muted/20 p-4">
              <Camera className="mt-0.5 size-5 shrink-0 text-primary" />
              <div className="min-w-0">
                <p className="text-sm font-medium">Kamera testi</p>
                <p className="mt-1 text-sm text-muted-foreground">
                  Sahifaga kirganda kamera avtomatik yoqiladi va jonli tasvir ko&apos;rsatiladi.
                </p>
              </div>
            </div>

            {devices.length > 0 ? (
              <div className="space-y-2">
                <Label htmlFor="camera-device">Kamera</Label>
                <select
                  id="camera-device"
                  className="flex h-10 w-full rounded-xl border border-input bg-background px-3 text-sm"
                  value={selectedDeviceId}
                  onChange={(event) => {
                    const nextDeviceId = event.target.value
                    setSelectedDeviceId(nextDeviceId)
                    void startCamera(nextDeviceId)
                  }}
                >
                  {devices.map((device) => (
                    <option key={device.deviceId} value={device.deviceId}>
                      {device.label}
                    </option>
                  ))}
                </select>
              </div>
            ) : null}

            {error ? (
              <div className="rounded-xl border border-destructive/30 bg-destructive/10 px-4 py-3 text-sm text-destructive">
                {error}
              </div>
            ) : null}
          </div>
        </Panel>
      </div>
    </PageContainer>
  )
}
