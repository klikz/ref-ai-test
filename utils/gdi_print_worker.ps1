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
        [int]$RotationDeg = 0
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

        $width = [int][Math]::Round($WidthMm / 25.4 * 100)
        $height = [int][Math]::Round($HeightMm / 25.4 * 100)
        if ($width -lt 1) { $width = 1 }
        if ($height -lt 1) { $height = 1 }

        $pd.DefaultPageSettings.Margins = New-Object System.Drawing.Printing.Margins(0, 0, 0, 0)
        $pd.OriginAtMargins = $false
        $paper = New-Object System.Drawing.Printing.PaperSize('LabelV2', $width, $height)
        $pd.DefaultPageSettings.PaperSize = $paper

        $script:pagesLeft = $Copies
        $handler = [System.Drawing.Printing.PrintPageEventHandler]{
            param($sender, $e)
            $g = $e.Graphics
            $g.PageUnit = [System.Drawing.GraphicsUnit]::Display
            $g.InterpolationMode = [System.Drawing.Drawing2D.InterpolationMode]::NearestNeighbor
            $g.PixelOffsetMode = [System.Drawing.Drawing2D.PixelOffsetMode]::Half
            if ($RotationDeg -eq 90) {
                $g.TranslateTransform($width / 2.0, $height / 2.0)
                $g.RotateTransform(90)
                $g.DrawImage($img, -$width / 2.0, -$height / 2.0, $width, $height)
            } else {
                $g.DrawImage($img, 0, 0, $width, $height)
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
    try {
        $job = $line | ConvertFrom-Json
        Invoke-LabelPrint `
            -ImagePath $job.ImagePath `
            -PrinterName $job.PrinterName `
            -WidthMm ([double]$job.WidthMm) `
            -HeightMm ([double]$job.HeightMm) `
            -Copies ([int]$job.Copies) `
            -DocumentName $job.DocumentName `
            -OutputPath $job.OutputPath `
            -RotationDeg ([int]$job.RotationDeg)
        [Console]::Out.WriteLine('OK')
    } catch {
        [Console]::Out.WriteLine(('ERR:' + $_.Exception.Message))
    }
    [Console]::Out.Flush()
}
