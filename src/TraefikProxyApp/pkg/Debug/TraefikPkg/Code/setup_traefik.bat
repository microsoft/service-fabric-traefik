@echo off
set LOG_OUTPUT=%TEMP%\setup_traefik.out

REM 
REM setup rules folder
REM Note: The Traefik image assumes the rules will be created under the subdirectory traefik/rules of the mounted folder. 
REM

set TRAEFIK_RULES_DIR=traefik\rules

if exist %TRAEFIK_RULES_DIR% (
  echo "%TRAEFIK_RULES_DIR% exists." >> %LOG_OUTPUT%
) else (
  echo "%TRAEFIK_RULES_DIR% does not exist. Creating directories..." >> %LOG_OUTPUT%
  md %TRAEFIK_RULES_DIR% >> %LOG_OUTPUT% 2>&1
)

REM
REM copy certificates in the work directory where the container can access it
REM Note: The Traefik image assumes the certs will be created under the subdirectory traefik/certs of the mounted folder. 
REM
set CONTAINER_TRAEFIK_CERT_DIR=traefik\certs

if "%TRAEFIK_CERT_THUMBPRINT%" NEQ "" (
  if exist %CONTAINER_TRAEFIK_CERT_DIR% ( 
    echo "%CONTAINER_TRAEFIK_CERT_DIR% exists." >> %LOG_OUTPUT%
  ) else (
    echo "%CONTAINER_TRAEFIK_CERT_DIR% does not exist. Creating directories..." >> %LOG_OUTPUT%
    md %CONTAINER_TRAEFIK_CERT_DIR% >> %LOG_OUTPUT% 2>&1
  )

REM    copy /y %TRAEFIK_CERT_DIR%\%TRAEFIK_CERT_THUMBPRINT%.crt %CONTAINER_TRAEFIK_CERT_DIR%\traefik.crt >> %LOG_OUTPUT% 2>&1
REM    copy /y %TRAEFIK_CERT_DIR%\%TRAEFIK_CERT_THUMBPRINT%.prv %CONTAINER_TRAEFIK_CERT_DIR%\traefik.prv >> %LOG_OUTPUT% 2>&1

REM    copy /y d:\traefik.crt %CONTAINER_TRAEFIK_CERT_DIR%\traefik.crt >> %LOG_OUTPUT% 2>&1
REM    copy /y d:\traefik.prv %CONTAINER_TRAEFIK_CERT_DIR%\traefik.prv >> %LOG_OUTPUT% 2>&1
) else (
  echo TRAEFIK_CERT_THUMBPRINT empty or not defined
)