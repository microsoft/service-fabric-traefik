#!/bin/bash
sfctl application delete --application-id Whoami
sfctl application unprovision --application-type-name WhoamiType --application-type-version 1.0.0
sfctl store delete --content-path Whoami
