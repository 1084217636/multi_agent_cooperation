# Groq Account Auto-Registration Script
# Uses PowerShell to control Edge browser

param(
    [string]$Email,
    [string]$Password
)

# Wait for browser to start
Start-Sleep -Seconds 2

# Create Internet Explorer object to control browser
$ie = New-Object -ComObject InternetExplorer.Application
$ie.Visible = $true

# Navigate to Groq console
$ie.Navigate("https://console.groq.com")

# Wait for page to load
while ($ie.Busy -or $ie.ReadyState -ne 4) {
    Start-Sleep -Milliseconds 100
}

Write-Host "Page loaded successfully"

# Wait for email input to appear
Start-Sleep -Seconds 2

# Enter email
$emailInput = $ie.Document.GetElementsByTagName("input") | Where-Object { $_.type -eq "email" }
if ($emailInput) {
    $emailInput.value = $Email
    Write-Host "Entered email: $Email"
} else {
    Write-Host "Email input not found"
    $ie.Quit()
    exit 1
}

# Click "Continue with email" button
$continueButton = $ie.Document.GetElementsByTagName("button") | Where-Object { $_.innerText -like "*Continue with email*" }
if ($continueButton) {
    $continueButton.Click()
    Write-Host "Clicked Continue with email button"
} else {
    Write-Host "Continue with email button not found"
    $ie.Quit()
    exit 1
}

# Wait for password page to load
Start-Sleep -Seconds 3

# Enter password
$passwordInput = $ie.Document.GetElementsByTagName("input") | Where-Object { $_.type -eq "password" }
if ($passwordInput) {
    $passwordInput.value = $Password
    Write-Host "Entered password"
} else {
    Write-Host "Password input not found"
    $ie.Quit()
    exit 1
}

# Click continue button
$continueButton2 = $ie.Document.GetElementsByTagName("button") | Where-Object { $_.innerText -like "*Continue*" }
if ($continueButton2) {
    $continueButton2.Click()
    Write-Host "Clicked Continue button"
} else {
    Write-Host "Continue button not found"
    $ie.Quit()
    exit 1
}

Write-Host "Registration request sent, please check email for confirmation"
