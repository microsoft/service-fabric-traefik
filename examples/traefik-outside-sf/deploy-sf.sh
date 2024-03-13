#!/bin/bash

# Variables
ResourceGroupName='traefik-cluster'
ClusterName='traefik-cluster'
Location='eastus'
Password='ChangemeIAMinsecureTraefik'
Subject='traefik-cluster.eastus.cloudapp.azure.com'
VaultName='traefik-cluster'
VmPassword='ChangemeIAMinsecureTraefik'
VmUserName='traefik'

# Login to Azure and set the subscription
az login

az account set --subscription $AZURE_SUBSCRIPTION

# Create resource group
az group create --name $ResourceGroupName --location $Location

# Create secure five node Linux cluster. Creates a key vault in a resource group
# and creates a certificate in the key vault. The certificate's subject name must match
# the domain that you use to access the Service Fabric cluster.  The certificate is downloaded locally.
az sf cluster create --resource-group $ResourceGroupName --location $Location --certificate-output-folder . --certificate-password $Password --certificate-subject-name $Subject --cluster-name $ClusterName --cluster-size 5 --os UbuntuServer1604 --vault-name $VaultName --vault-rg $ResourceGroupName --vm-password $VmPassword --vm-user-name $VmUserName

# Create traefik VM
az vm create \
    --name traefik \
    --resource-group $ResourceGroupName \
    --size Standard_B2s \
    --image Ubuntu2204 \
    --admin-username $VmUserName --admin-password $VmPassword --vnet-name=VNet --subnet=Subnet-0 --custom-data ./vm-custom-data.sh

# Open port for traefik VM
az vm open-port --port 80 --resource-group $ResourceGroupName --name traefik --priority 900
az vm open-port --port 8080 --resource-group $ResourceGroupName --name traefik --priority 901

# Generate cert and key for traefik
openssl pkcs12 -info -in traefik-clustereastuscloudappazurecom.pfx  -nodes -nocerts -out ./traefik/compose/certs/sf.key
openssl pkcs12 -in traefik-clustereastuscloudappazurecom.pfx -clcerts -nokeys -out ./traefik/compose/certs/sf.crt
