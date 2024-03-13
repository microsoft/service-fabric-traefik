#!/bin/bash

# Variables
ResourceGroupName='traefik-cluster'

# Login to Azure and set the subscription
az login
az account set --subscription $AZURE_SUBSCRIPTION

# Delete all resources created in the resource group
az group delete --name $ResourceGroupName

# Purge the vault
az keyvault purge --name $ResourceGroupName
