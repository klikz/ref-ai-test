Add-Type -AssemblyName System.Drawing
Add-Type -AssemblyName System.Drawing.Printing

function Invoke-LabelPrint {
    param(
        [string]$ImagePath,
        [string]$PrinterName,
        [double]$WidthMm,
        [double]$HeightMm,
        [int]$Copies,
        [string]$DocumentName,
        [string]$OutputPath,
        [int]$RotationDeg = 0,
        [bool]$ForcePaperSize = $false
    )

    if ($Copies -lt 1) { $Copies = 1 }

    $img = [System.Drawing.Image]::FromFile($ImagePath)
    try {
        $pd = New-Object System.Drawing.Printing.PrintDocument
        $pd.PrintController = New-Object System.Drawing.Printing.StandardPrintController
        $pd.PrinterSettings.PrinterName = $PrinterName
        $pd.DocumentName = $DocumentName
        if (-not [string]::IsNullOrWhiteSpace($OutputPath)) {
            if ($PrinterName -ne 'Microsoft Print to PDF') {
                throw 'OutputPath faqat Microsoft Print to PDF uchun ishlatiladi'
            }
            $pd.PrinterSettings.PrintToFile = $true
            $pd.PrinterSettings.PrintFileName = $OutputPath
        }

        # Always zero margins: default ~1" MarginBounds shrinks content on small labels.
        $pd.DefaultPageSettings.Margins = New-Object System.Drawing.Printing.Margins(0, 0, 0, 0)
        $pd.OriginAtMargins = $false
        $pd.DefaultPageSettings.Landscape = $false

        # Caller passes physical media size (already swapped for album/90°).
        # Must set PaperSize — otherwise PDF/default page stays A4 and only content looks rotated.
        if ($WidthMm -gt 0 -and $HeightMm -gt 0) {
            $wantW = [int][Math]::Round($WidthMm / 25.4 * 100)
            $wantH = [int][Math]::Round($HeightMm / 25.4 * 100)
            if ($wantW -lt 1) { $wantW = 1 }
            if ($wantH -lt 1) { $wantH = 1 }

            $tol = 20 # ~5mm
            $matched = $null
            foreach ($candidate in @($pd.PrinterSettings.PaperSizes)) {
                $ok = (
                    [Math]::Abs([int]$candidate.Width - $wantW) -le $tol -and
                    [Math]::Abs([int]$candidate.Height - $wantH) -le $tol
                )
                if ($ok) {
                    $matched = $candidate
                    break
                }
            }
            if ($null -ne $matched) {
                $pd.DefaultPageSettings.PaperSize = $matched
            } elseif ($ForcePaperSize -eq $true -or -not [string]::IsNullOrWhiteSpace($OutputPath) -or $PrinterName -eq 'Microsoft Print to PDF') {
                # Custom size for PDF/generic only. Thermal drivers: avoid custom PaperSize
                # (resets private DEVMODE — darkness/speed fall back to factory defaults).
                $custom = New-Object System.Drawing.Printing.PaperSize('ACLabel', $wantW, $wantH)
                [void]$pd.PrinterSettings.PaperSizes.Add($custom)
                $pd.DefaultPageSettings.PaperSize = $custom
            }
            # else: keep driver default paper; image still drawn into PageBounds
        }

        $script:pagesLeft = $Copies
        $handler = [System.Drawing.Printing.PrintPageEventHandler]{
            param($sender, $e)
            $g = $e.Graphics
            $g.PageUnit = [System.Drawing.GraphicsUnit]::Display
            $g.InterpolationMode = [System.Drawing.Drawing2D.InterpolationMode]::NearestNeighbor
            $g.PixelOffsetMode = [System.Drawing.Drawing2D.PixelOffsetMode]::Half

            $bounds = $e.PageBounds
            if ($bounds.Width -lt 1 -or $bounds.Height -lt 1) {
                $bounds = $e.MarginBounds
            }

            # Image is pre-rotated in Go for 90°; draw 1:1 into media bounds (no stretch-rotate).
            if ($RotationDeg -eq 90) {
                # Legacy fallback if caller still sends unrotated bitmap + RotationDeg=90.
                $cx = $bounds.X + $bounds.Width / 2.0
                $cy = $bounds.Y + $bounds.Height / 2.0
                $g.TranslateTransform($cx, $cy)
                $g.RotateTransform(90)
                $g.DrawImage($img, -$bounds.Height / 2.0, -$bounds.Width / 2.0, $bounds.Height, $bounds.Width)
            } else {
                $g.DrawImage($img, $bounds.X, $bounds.Y, $bounds.Width, $bounds.Height)
            }
            $script:pagesLeft--
            $e.HasMorePages = ($script:pagesLeft -gt 0)
        }

        $pd.add_PrintPage($handler)
        $pd.Print()
    } finally {
        if ($img) { $img.Dispose() }
    }
}

[Console]::Out.WriteLine('READY')
[Console]::Out.Flush()

while ($true) {
    $line = [Console]::In.ReadLine()
    if ($null -eq $line) { break }
    if ($line -eq 'EXIT') { break }
    # Replies echo ReqId so the client can drop a reply to an abandoned request
    # instead of restarting this worker over one desynced line.
    $reqId = 0
    try {
        $job = $line | ConvertFrom-Json
        if ($null -ne $job.ReqId) { $reqId = [uint64]$job.ReqId }
        Invoke-LabelPrint `
            -ImagePath $job.ImagePath `
            -PrinterName $job.PrinterName `
            -WidthMm ([double]$job.WidthMm) `
            -HeightMm ([double]$job.HeightMm) `
            -Copies ([int]$job.Copies) `
            -DocumentName $job.DocumentName `
            -OutputPath $job.OutputPath `
            -RotationDeg ([int]$job.RotationDeg) `
            -ForcePaperSize ([bool]$job.ForcePaperSize)
        [Console]::Out.WriteLine("OK $reqId 0")
    } catch {
        $msg = ($_.Exception.Message -replace '[\r\n]+', ' ')
        [Console]::Out.WriteLine("ERR $reqId $msg")
    }
    [Console]::Out.Flush()
}
