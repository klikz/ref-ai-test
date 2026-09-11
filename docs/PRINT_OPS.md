# Print operations checklist (V2)

## Print processes

| Til | Process | Funksiya |
|-----|---------|----------|
| **GDI** | PowerShell `gdi_print_worker.ps1` | prepare → GDI worker submit |
| **TSPL** | API child: `--print-worker=tspl` | prepare (payload) → Win32 RAW worker |
| **ZPL** | API child: `--print-worker=zpl` | prepare (payload) → Win32 RAW worker |
| **AC-FILE*** | Fayl (Windows queue yo‘q) | ZPL/TSPL payload quriladi → `print_preview/*.png` (+ `.zpl`/`.tspl`) |

Test printerlar: `AC-FILE-ZPL` (T3), `AC-FILE-TSPL` (I1). Papka: `LABEL_FILE_OUTPUT_DIR` yoki default `print_preview/`.

## Concurrency modeli

Render va spoolga yuborish **alohida** cheklanadi. Ilgari ikkalasi bitta til
semaforasini bo‘lishardi, shuning uchun sekin render printer slotini band qilar,
band printer esa boshqa liniyalarning renderini to‘xtatib turardi.

| Bosqich | Cheklov | Env |
|---------|---------|-----|
| Render (rasm + payload) | render semaforasi | `PRINT_V2_RENDER_WORKERS` (default: CPU soni; `PRINT_V2_MAX_WORKERS` fallback) |
| Spoolga yuborish | har til uchun worker jarayonlari pooli | `PRINT_V2_TSPL_WORKERS` / `PRINT_V2_ZPL_WORKERS` (default **4**), `PRINT_V2_GDI_WORKERS` (default **2**) |

Har til uchun bitta worker jarayoni o‘rniga **pool** ishlatiladi: ilgari bitta
mutex butun zavoddagi shu tildagi barcha printerlarni ketma-ket qilar, ya’ni
bitta sekin printer qolganlarini ham to‘xtatardi.

Bo‘sh worker kutish limiti: `PRINT_V2_SUBMIT_QUEUE_TIMEOUT_MS` (default 15000).
Limit tugasa handler cheksiz bloklanmaydi, «print navbati band» xatosi qaytadi.

Workerlar `StartPrintWorkers()` orqali server startida oldindan ishga tushiriladi
(sovuq start bir necha soniya turadi).

## Retry siyosati (duplikatga qarshi)

Render+payload **bir marta**; qayta urinish faqat spool submit + verify.
Urinishlar soni: **2**.

Qayta urinish faqat printerga **bayt ketmagani aniq** bo‘lgan xatolarda:

| Holat | Qayta urinadimi | Sabab |
|-------|-----------------|-------|
| Worker jarayoni o‘lgan / ishga tushmagan | ✅ ha | Spoolerga hech narsa bormagan |
| Job navbatda `Error`/`Paused`/`Offline` — bekor qilindi | ✅ ha | Bloklangan job chop etilmagan |
| Worker javob timeout | ❌ yo‘q | Etiketka chiqqan bo‘lishi mumkin |
| Worker noma‘lum javob / protokol buzilishi | ❌ yo‘q | Natija noaniq |
| GDI submit xatosi | ❌ yo‘q | Natija noaniq |

Ilgari har qanday xatoda 3 marta urinilardi, bu esa qotib qolgan printerda
bir nechta bir xil etiketka chiqishiga olib kelardi.

## Worker IPC

Length-prefix binary frame (JSON header + raw payload; base64 yo‘q).
Javob bir qator: `OK <reqID> <jobID>` yoki `ERR <reqID> <xabar>`.

`reqID` tashlab yuborilgan so‘rovning kechikkan javobini ajratish uchun: usiz
bitta desinxronlashgan qator keyingi barcha javoblarni siljitib, workerni
qayta ishga tushirishga majbur qilardi.

Worker javob timeout: `PRINT_V2_WORKER_TIMEOUT_MS` (default 30000) — timeoutda child restart.
## Production til tanlash

| Brand | Til | Sozlamalar |
|-------|-----|------------|
| Zebra | **ZPL** | Shablon density/speed (`~SD` / `^PR`) |
| Gprinter / Gainscha / XPrinter / TSC | **TSPL** | Shablon density/speed/gap |
| Ofis / PDF | **GDI** | Windows Printing **Defaults** |

GDI ni Zebra/Gprinter production uchun asosiy yo‘l qilmang.

## Windows service / user

1. API ni printerlar o‘rnatilgan Windows user bilan ishga tushiring.
2. Sozlamalarni **Printing Defaults** (barcha userlar) ga saqlang, faqat Preferences emas.
3. GDI («Use defaults») ishlatilsa: Settings o‘zgargach bir marta Windows Test Print yoki API restart.

## Shablon sozlamalari

1. Label template editor → Density / Speed / Gap.
2. **Use defaults**:
   - **GDI** printer → Windows Printing Preferences.
   - **ZPL/TSPL** printer → RAW til saqlanadi; density/speed jobga yozilmaydi (firmware). TSPL da GAP saqlanadi.
3. **Faqat o‘lcham (size only)** → ZPL/TSPL da SIZE / ^PW/^LL; density/speed yozilmaydi (TSPL GAP saqlanadi).
4. Ikkalasi o‘chirilsa → ZPL/TSPL da density/speed/gap jobga yoziladi.

GDI ishlatilsa: Settings o‘zgargach bir marta Windows Test Print yoki API restart.

## Tekshiruv

1. `/printers-v2` → qator **Test** tugmasi.
2. Media calibrate, DPI 203 yoki 300, dithering off (GDI).
3. Concurrency env’lari — yuqoridagi «Concurrency modeli» bo‘limiga qarang.
4. Spool verify (Windows, Win32 `EnumJobs`/`SetJob`; PowerShell fallback):
   - Job **spooler JobId** bo‘yicha kuzatiladi (hujjat nomi bo‘yicha emas).
     Nom bo‘yicha moslashtirish qayta chop etishda eski `Retained` jobni
     tanlab, noto‘g‘ri xato berardi.
   - Navbatdan chiqdi → chop etilgan, OK
   - Stuck (`Error`/`Paused`/`Offline`/…) → **faqat o‘sha job** bekor + retry
   - Hali `Spooling`/`Printing` → **bekor qilinmaydi**, fon kuzatuvchisiga
     topshiriladi. Ilgari timeout’dan keyin faol job bekor qilinardi — bu sekin
     printerlarda soxta «navbatda qolib ketdi» xatosi va dublikat chiqarardi.
   - O‘chirish: `PRINT_V2_SPOOL_VERIFY=0`
   - Sinxron oyna: `PRINT_V2_SPOOL_VERIFY_MS` (default **400**, ilgari 3000)
   - Worker reply: `PRINT_V2_WORKER_TIMEOUT_MS` (default 30000)
5. Fon spool kuzatuvchisi — operator «chop etildi» javobini olgandan keyin:
   - Job keyinroq qotib qolsa → bekor qilinadi va `stage=spool_late` event
     yoziladi. **Qayta chop etilmaydi**: etiketka allaqachon chiqqan bo‘lishi
     mumkin, shuning uchun avtomatik takror dublikat xavfini tug‘diradi.
   - Kuzatish muddati: `PRINT_V2_SPOOL_WATCH_MS` (default 60000). Muddat
     tugaganda hali harakatlanayotgan job bekor qilinmaydi, faqat WARN yoziladi.
   - Tekshirish oralig‘i: `PRINT_V2_SPOOL_WATCH_INTERVAL_MS` (default 1000)
6. Printdan oldin navbat tozalash faqat **eski** (20 soniyadan katta) va
   kuzatilmayotgan stuck joblarni o‘chiradi — parallel printning jobiga tegmaydi.
7. Muammoli print loglari (`component=print_v2`):
   - `WARN` — attempt fail, spool stuck, GDI+thermal, recovery after retry
   - `ERR` — yakuniy fail (+ `queue_snapshot`)
   - Default `LOG_LEVEL=info` da ham ko‘rinadi (`debug` shart emas)
   - `logger.log` va konsol

## Metrics

UI: `/printers-v2/metrics` — sana filtri, xato detallari, oralik reset.

DB: `lines.print_v2_events` (migration `112_print_v2_events.sql`).
Label template print settings: `density` / `speed` / `gap_mm` / `use_printer_defaults` / `size_only` (migration `113_label_template_print_settings.sql`).

`meta` ustunidagi bosqich taymerlari — sekin printni render yoki spoolerga
bog‘lash uchun:

| Kalit | Ma’nosi |
|-------|---------|
| `render_ms` | Rasm + payload qurish (render semaforasini kutish bilan) |
| `submit_ms` | Spoolerga yuborish + verify (barcha urinishlar yig‘indisi) |
| `spool_job_id` | Windows spooler job raqami — navbatda topish uchun |
| `attempts` | Ishlatilgan urinishlar soni |

`stage` qiymatlari: `ok`, `render`, `spool`, `gdi`, `spool_late` (fon
kuzatuvchisi topgan kech xatolik).

## Shrift keshi

Etiketka renderi shriftni har chizishda diskdan qayta o‘qimaydi: parse qilingan
shrift global keshda, `font.Face` esa har render uchun alohida (truetype face
o‘zgaruvchan glyph keshi saqlaydi, shuning uchun uni goroutine’lar orasida
bo‘lishib bo‘lmaydi).

Jadvalli etiketkada bu 17.2 ms → 4.4 ms va 64.5 MB → 7.2 MB allocation berdi.

Render natijasi `utils/testdata/golden_*.png` bilan piksel darajasida
qulflangan. Shrift yoki layout kodini o‘zgartirgandan keyin:

```bash
go test ./utils -run TestLabelRenderGolden
```

Windows shriftlari boshqa versiyada bo‘lsa golden’ni bir marta yangilang:

```bash
go test ./utils -run TestLabelRenderGolden -update-golden
```

## Label template keshi

Shablon har printda bir necha marta o‘qilardi (handler + print helper).
Endi repo darajasida keshlanadi, yozishda darhol bekor qilinadi.

TTL: `LABEL_TEMPLATE_CACHE_TTL_MS` (default 15000). `0` — keshni butunlay
o‘chiradi. TTL faqat bu jarayondan tashqarida (masalan to‘g‘ridan-to‘g‘ri
bazada) qilingan tahrirlar uchun himoya.

## HTTP server timeout’lari

| Env | Default | Izoh |
|-----|---------|------|
| `HTTP_READ_HEADER_TIMEOUT_S` | 15 | Slowloris himoyasi |
| `HTTP_READ_TIMEOUT_S` | 60 | Body o‘qish (rasm upload uchun yetarli) |
| `HTTP_IDLE_TIMEOUT_S` | 120 | Keep-alive |

`WriteTimeout` **ataylab o‘rnatilmagan**: T3 serial WebSocket va kamera MJPEG
stream uzoq muddatli, write deadline ularni oqim o‘rtasida uzib qo‘yardi.

API:
- `POST /api/tech/printers-v2/metrics` — body: `date_from`, `date_to` (YYYY-MM-DD). Javob: jami + `lines[]` (har liniya: `success`, `fail`, `total_ms`, `last_ms`); `in_flight` process ichida.
- `POST /api/tech/printers-v2/metrics/errors` — xatolar ro‘yxati (`error_message`, `error_detail`, meta).
- `POST /api/tech/printers-v2/metrics/reset` — tanlangan oralikdagi eventlarni o‘chirish.
