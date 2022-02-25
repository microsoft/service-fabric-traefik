if not exist ..\work\dyn md ..\work\dyn
copy dynConfig\dyn.yaml ..\work

traefik.exe --entryPoints.web.address=:%TRAEFIK_HTTP_PORT% --providers.file.watch=true --providers.file.directory="..\work" --log.level=debug --api.dashboard=%TRAEFIK_ENABLE_DASHBOARD%
