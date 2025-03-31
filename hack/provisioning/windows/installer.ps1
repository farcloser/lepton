# Get the HNS module
$dirPath = (New-Item -Path "$(Split-Path $PROFILE.CurrentUserCurrentHost)/Modules/HNS" -ItemType Directory -Force).FullName
$Uri = 'https://raw.githubusercontent.com/microsoft/SDN/dd4e8708ed184b49d3fddd611b6027f1755c6edb/Kubernetes/windows/hns.v2.psm1'
Invoke-WebRequest -Uri $Uri -OutFile "$dirPath/hns.psm1"
Get-Module -Name HNS -ListAvailable -Refresh
Import-Module HNS

# Get the toolkit
mkdir -force "$Env:ProgramFiles\containers-toolkit"
git clone --quiet --branch main https://github.com/microsoft/containers-toolkit.git $Env:ProgramFiles\containers-toolkit
cd $Env:ProgramFiles\containers-toolkit
git checkout 254e82bd15b83380d720f1a12768f10b99897b81
$env:PSModulePath += ";$Env:ProgramFiles\containers-toolkit"
Get-Module -Name "containers-toolkit" -ListAvailable -Refresh
Import-Module -Name containers-toolkit -Force

# FIXME: when published in the gallery
# Install-Module -Name Containers-Toolkit

# Check Hyper V is up and running
Get-WindowsOptionalFeature -Online | `
    Where-Object { $_.FeatureName -match "Microsoft-Hyper-V(-All)?$" } | `
    Select-Object FeatureName, Possible, State, RestartNeeded

# Install containerd
Install-Containerd -Version "$env:CONTAINERD_VERSION" -Setup -Force

# Install WinCNI
Install-WinCNIPlugin -WinCNIVersion "$env:WINCNI_VERSION" -Force
$gateway = (Get-NetIPAddress -InterfaceAlias 'vEthernet (nat)' -AddressFamily IPv4).IPAddress
$cidr = (Get-NetIPAddress -InterfaceAlias 'vEthernet (nat)' -AddressFamily IPv4).PrefixLength
Initialize-NatNetwork -Gateway $gateway -CIDR $cidr

# Install buildkitd
Install-BuildKit -Version "$env:BUILDKIT_VERSION" -Setup -Force
