param (
    [Parameter(Position=0, Mandatory=$true)]
    [string]$Action,
    
    [Parameter()]
    [string]$SSID,
    
    [Parameter()]
    [string]$Password,
    
    [Parameter()]
    [switch]$Enable
)

# Set output encoding to UTF-8
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8
$PSDefaultParameterValues['*:Encoding'] = 'utf8'

# Check if running as administrator
$isAdmin = ([Security.Principal.WindowsPrincipal][Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
if (-not $isAdmin) {
    @{
        Success = $false
        Error = "This script requires administrator privileges"
    } | ConvertTo-Json
    exit 1
}

# Load Windows Runtime assemblies
try {
    Add-Type -AssemblyName System.Runtime.WindowsRuntime
    
    # Helper function to await WinRT async operations
    function Await {
        param(
            [object]$WinRtTask,
            [Type]$ResultType
        )
        
        $asTaskGeneric = ([System.WindowsRuntimeSystemExtensions].GetMethods() | 
            Where-Object { 
                $_.Name -eq 'AsTask' -and 
                $_.GetParameters().Count -eq 1 -and 
                $_.GetParameters()[0].ParameterType.Name -eq 'IAsyncOperation`1' 
            })[0]
        
        $asTask = $asTaskGeneric.MakeGenericMethod($ResultType)
        $netTask = $asTask.Invoke($null, @($WinRtTask))
        $netTask.Wait(-1) | Out-Null
        return $netTask.Result
    }

    # Get TetheringManager instance
    function Get-TetheringManager {
        try {
            $connectionProfile = [Windows.Networking.Connectivity.NetworkInformation,Windows.Networking.Connectivity,ContentType=WindowsRuntime]::GetInternetConnectionProfile()
            if ($null -eq $connectionProfile) {
                throw "No active internet connection found"
            }
            
            $tetheringManager = [Windows.Networking.NetworkOperators.NetworkOperatorTetheringManager,Windows.Networking.NetworkOperators,ContentType=WindowsRuntime]::CreateFromConnectionProfile($connectionProfile)
            if ($null -eq $tetheringManager) {
                throw "Failed to create tethering manager"
            }
            
            return $tetheringManager
        }
        catch {
            throw "Failed to initialize tethering manager: $_"
        }
    }
}
catch {
    @{
        Success = $false
        Error = "Failed to load Windows Runtime assemblies: $_"
    } | ConvertTo-Json
    exit 1
}

# Function to get hotspot status
function Get-HotspotStatus {
    try {
        $tetheringManager = Get-TetheringManager
        $config = $tetheringManager.GetCurrentAccessPointConfiguration()
        $state = $tetheringManager.TetheringOperationalState
        $clients = $tetheringManager.GetTetheringClients()

        @{
            Success = $true
            Enabled = $state -eq 1  # 1 means "On"
            SSID = $config.Ssid
            ClientsCount = $clients.Count
            Authentication = $config.Authentication
            Encryption = $config.Encryption
            MaxClientCount = $config.MaxClientCount
        } | ConvertTo-Json
    }
    catch {
        @{
            Success = $false
            Error = $_.Exception.Message
        } | ConvertTo-Json
        exit 1
    }
}

# Function to configure hotspot
function Set-HotspotConfig {
    param (
        [string]$SSID,
        [string]$Password,
        [bool]$Enable
    )
    
    try {
        $tetheringManager = Get-TetheringManager
        
        # Create new configuration
        $config = New-Object Windows.Networking.NetworkOperators.NetworkOperatorTetheringAccessPointConfiguration
        $config.Ssid = $SSID
        $config.Passphrase = $Password
        # MaxClientCount is not available in Windows 10/11 API
        
        # Configure the access point
        $operation = $tetheringManager.ConfigureAccessPointAsync($config)
        Await -WinRtTask $operation -ResultType ([Windows.Networking.NetworkOperators.NetworkOperatorTetheringOperationResult])
        
        if ($Enable) {
            $operation = $tetheringManager.StartTetheringAsync()
            Await -WinRtTask $operation -ResultType ([Windows.Networking.NetworkOperators.NetworkOperatorTetheringOperationResult])
        }
        
        @{
            Success = $true
            Message = "Hotspot configured successfully"
        } | ConvertTo-Json
    }
    catch {
        @{
            Success = $false
            Error = $_.Exception.Message
        } | ConvertTo-Json
        exit 1
    }
}

# Function to set hotspot status
function Set-HotspotStatus {
    param (
        [bool]$Enable
    )
    
    try {
        $tetheringManager = Get-TetheringManager
        
        if ($Enable) {
            $operation = $tetheringManager.StartTetheringAsync()
            $result = Await -WinRtTask $operation -ResultType ([Windows.Networking.NetworkOperators.NetworkOperatorTetheringOperationResult])
            $message = "Hotspot enabled"
        } else {
            $operation = $tetheringManager.StopTetheringAsync()
            $result = Await -WinRtTask $operation -ResultType ([Windows.Networking.NetworkOperators.NetworkOperatorTetheringOperationResult])
            $message = "Hotspot disabled"
        }
        
        @{
            Success = $true
            Message = $message
        } | ConvertTo-Json
    }
    catch {
        @{
            Success = $false
            Error = $_.Exception.Message
        } | ConvertTo-Json
        exit 1
    }
}

# Execute based on action
switch ($Action) {
    "status" {
        Get-HotspotStatus
    }
    
    "configure" {
        if ([string]::IsNullOrEmpty($SSID)) {
            @{
                Success = $false
                Error = "SSID is required"
            } | ConvertTo-Json
            exit 1
        }
        if ($SSID.Length -gt 32) {
            @{
                Success = $false
                Error = "SSID length must be 32 characters or less"
            } | ConvertTo-Json
            exit 1
        }
        if ([string]::IsNullOrEmpty($Password)) {
            @{
                Success = $false
                Error = "Password is required"
            } | ConvertTo-Json
            exit 1
        }
        if ($Password.Length -lt 8 -or $Password.Length -gt 63) {
            @{
                Success = $false
                Error = "Password length must be between 8 and 63 characters"
            } | ConvertTo-Json
            exit 1
        }
        
        Set-HotspotConfig -SSID $SSID -Password $Password -MaxClients $MaxClients -Enable $Enable
    }
    
    "enable" {
        Set-HotspotStatus -Enable $true
    }
    
    "disable" {
        Set-HotspotStatus -Enable $false
    }
    
    default {
        @{
            Success = $false
            Error = "Unknown action: $Action"
        } | ConvertTo-Json
        exit 1
    }
}