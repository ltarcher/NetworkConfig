$PSVersionTable
"-" * 100

Add-Type -AssemblyName System.Runtime.WindowsRuntime
$asTaskGeneric = ([System.WindowsRuntimeSystemExtensions].GetMethods() | Where-Object { $_.Name -eq 'AsTask' -and $_.GetParameters().Count -eq 1 -and $_.GetParameters()[0].ParameterType.Name -eq 'IAsyncOperation`1' })[0]

Function Await($WinRtTask, $ResultType) {
    $asTask = $asTaskGeneric.MakeGenericMethod($ResultType)
    $netTask = $asTask.Invoke($null, @($WinRtTask))
    $netTask.Wait(-1) | Out-Null
    $netTask.Result
}

Function AwaitAction($WinRtAction) {
    $asTask = ([System.WindowsRuntimeSystemExtensions].GetMethods() | Where-Object { $_.Name -eq 'AsTask' -and $_.GetParameters().Count -eq 1 -and !$_.IsGenericMethod })[0]
    $netTask = $asTask.Invoke($null, @($WinRtAction))
    $netTask.Wait(-1) | Out-Null
}

Function Get_TetheringManager() {
    $connectionProfile = [Windows.Networking.Connectivity.NetworkInformation,Windows.Networking.Connectivity,ContentType=WindowsRuntime]::GetInternetConnectionProfile()
    $tetheringManager = [Windows.Networking.NetworkOperators.NetworkOperatorTetheringManager,Windows.Networking.NetworkOperators,ContentType=WindowsRuntime]::CreateFromConnectionProfile($connectionProfile)
    return $tetheringManager;
}

Function SetHotspot($Enable) {
    $tetheringManager = Get_TetheringManager

    if ($Enable -eq 1) {
        if ($tetheringManager.TetheringOperationalState -eq 1)
        {
            "Hotspot is already On!"
        }
        else{
            "Hotspot is off! Turning it on"
            Await ($tetheringManager.StartTetheringAsync()) ([Windows.Networking.NetworkOperators.NetworkOperatorTetheringOperationResult])
        }
    }
    else {
        if ($tetheringManager.TetheringOperationalState -eq 1)
        {
            "Hotspot is on! Turning it off"
            Await ($tetheringManager.StopTetheringAsync()) ([Windows.Networking.NetworkOperators.NetworkOperatorTetheringOperationResult])
        }
        else{
            "Hotspot is already Off!"
        }
    }
}

# Define a function to check the status of the hotspot
Function Check_HotspotStatus() {
    $tetheringManager = Get_TetheringManager
    return $tetheringManager.TetheringOperationalState -eq "Off"
}

# Define a function to start the hotspot
Function Start_Hotspot() {
    $tetheringManager = Get_TetheringManager
    Await ($tetheringManager.StartTetheringAsync()) ([Windows.Networking.NetworkOperators.NetworkOperatorTetheringOperationResult])
}

Function exitCountdown($sec)
{
    for (; $sec -ge 0; --$sec)
    {
        "$sec"
        Start-Sleep -Seconds 1
    }
    exit 0
}

if ($args.Length -eq 0) {
    while (Check_HotspotStatus) {
        SetHotspot 1
        Start-Sleep -Seconds 2
        if (Check_HotspotStatus)
        {
            "Failure.Try again in 2s."
            Start-Sleep -Seconds 2
            continue
        }
        else
        {
            "Success.Exit in 10s."
            exitCountdown 10
            exit 0
        }
    }

    "Hotspot is already.Exit in 10s."
    exitCountdown 10
}
else {
    switch ($args[0]) {
        "0" {
            SetHotspot 0
            break
        }
        "1" {
            SetHotspot 1
            break
        }
        default {
            "Invalid parameter, please enter 1 to turn on hotspot, enter 0 to turn off hotspot"
            exit 1
        }
    }
}
