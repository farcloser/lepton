mkdir -force "$Env:ProgramFiles\containers-toolkit"
# FIXME: pin this.
git clone --quiet --depth 1 --branch main https://github.com/microsoft/containers-toolkit.git $Env:ProgramFiles\containers-toolkit
$env:PSModulePath += ";$Env:ProgramFiles\containers-toolkit"
Get-Module -Name "containers-toolkit" -ListAvailable
Import-Module -Name containers-toolkit -Force

# Containerd
Install-Containerd -Version "$env:CONTAINERD_VERSION" -Setup

# WinCNI
Install-WinCNIPlugin -WinCNIVersion "$env:WINCNI_VERSION"
$gateway = (Get-NetIPAddress -InterfaceAlias 'vEthernet (nat)' -AddressFamily IPv4).IPAddress
$cidr = (Get-NetIPAddress -InterfaceAlias 'vEthernet (nat)' -AddressFamily IPv4).PrefixLength
Initialize-NatNetwork -Gateway $gateway -CIDR $cidr

# Buildkit
Install-BuildKit -Version "$env:BUILDKIT_VERSION" -Setup
