# Hailo Standard Library

This is Hailo's own version of the Go standard library.

See https://hailo.jira.com/wiki/display/HTWO/2014/02/25/Shared+Code for details of how this library is managed.

Next release candidate branch is rc20140104


Every service using the localisation library that accesses the hob info: ```GetHob(hob)```, ```GetServiceType(hob)``` or ```GetServiceTypes(hob)``` will have to add the following service-2-service permissions
```
execute grantservice {"endpoint":  {"service": "com.HailoOSS.service.hob", "endpoint": "multiconfig", "granted": [ {"name": "your.service.name", "role": "ADMIN" }]}}
execute grantservice {"endpoint":  {"service": "com.HailoOSS.service.hob", "endpoint": "readhob", "granted": [ {"name": "your.service.name", "role": "ADMIN" }]}}
execute grantservice {"endpoint":  {"service": "com.HailoOSS.service.hob", "endpoint": "readservicetypes", "granted": [ {"name": "your.service.name", "role": "ADMIN" }]}}
```
