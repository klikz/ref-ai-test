param(
    [Parameter(Mandatory = $true)][string]$ImagePath,
    [Parameter(Mandatory = $true)][string]$PrinterName,
    [Parameter(Mandatory = $true)][double]$WidthMm,
    [Parameter(Mandatory = $true)][double]$HeightMm,
    [int]$Copies = 1,
    [string]$DocumentName = "LabelV2"
)

if ($Copies -lt 1) { $Copies = 1 }

Add-Type -AssemblyName System.Drawing
Add-Type -AssemblyName System.Drawing.Printing

$img = [System.Drawing.Image]::FromFile($ImagePath)
try {
    $pd = New-Object System.Drawing.Printing.PrintDocument
    $pd.PrintController = New-Object System.Drawing.Printing.StandardPrintController
    $pd.PrinterSettings.PrinterName = $PrinterName
    $pd.DocumentName = $DocumentName

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
        $g.DrawImage($img, 0, 0, $width, $height)
        $script:pagesLeft--
        $e.HasMorePages = ($script:pagesLeft -gt 0)
    }

    $pd.add_PrintPage($handler)
    $pd.Print()
} finally {
    if ($img) {
        $img.Dispose()
    }
}
