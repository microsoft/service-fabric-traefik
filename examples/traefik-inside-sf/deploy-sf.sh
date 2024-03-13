#!/bin/bash

# Variables
ResourceGroupName='traefik-cluster'
ClusterName='traefik-cluster'
Location='eastus'
Password="${SF_PASSWORD}"
Subject='traefik-cluster.eastus.cloudapp.azure.com'
VaultName='traefik-cluster'
VmUserName='traefik'
VmPassword="${SF_PASSWORD}"

# Login to Azure and set the subscription
az login

az account set --subscription $AZURE_SUBSCRIPTION

# Create resource group
az group create --name $ResourceGroupName --location $Location

# Create secure five node Linux cluster. Creates a key vault in a resource group
# and creates a certificate in the key vault. The certificate's subject name must match
# the domain that you use to access the Service Fabric cluster.  The certificate is downloaded locally.
az sf cluster create --resource-group $ResourceGroupName --location $Location --certificate-output-folder . --certificate-password $Password --certificate-subject-name $Subject --cluster-name $ClusterName --cluster-size 5 --os UbuntuServer1604 --vault-name $VaultName --vault-rg $ResourceGroupName --vm-password $VmPassword --vm-user-name $VmUserName

# Extract SF cert and key for traefik.
echo "Extracting Service Fabric cert and key for Traefik App. There is no import password, you can press RETURN"
openssl pkcs12 -info -in "${ClusterName}eastuscloudappazurecom.pfx" -nodes -nocerts -out ./Traefik/TraefikPkg/Fetcher.Code/certs/sf.key
openssl pkcs12 -in "${ClusterName}eastuscloudappazurecom.pfx" -clcerts -nokeys -out ./Traefik/TraefikPkg/Fetcher.Code/certs/sf.crt
