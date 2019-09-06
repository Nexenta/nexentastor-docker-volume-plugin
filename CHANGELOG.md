
<a name="v1.1.0"></a>
## [v1.1.0](https://github.com/Nexenta/nexentastor-docker-volume-plugin/compare/v1.0.1...v1.1.0) (2019-09-03)

### Bug Fixes

* pre-compile regexp's

### Features

* NEX-20838 - add timeo=100 to default options for NFS mounts


<a name="v1.0.1"></a>
## [v1.0.1](https://github.com/Nexenta/nexentastor-docker-volume-plugin/compare/v1.0.0...v1.0.1) (2019-05-07)

### Bug Fixes

* NEX-20603 - volume list shows only 100 first volumes


<a name="v1.0.0"></a>
## v1.0.0 (2019-04-22)

### Bug Fixes

* fs export path update leaves unmounted volumes on docker host
* NEX-20385 - treat vers=4 and vers=4.0 as same versions
* NEX-20385 - do not check mount point source on docker volume mount
* NEX-20385 - do not remove ns filesystem on 'docker volume remove'
* NEX-13886 - use the same volume in more then one container
* don't return VolumeDriver/Get error if there is no such volume, fix makefile
* log all errors before response

### Pull Requests

* Merge pull request [#2](https://github.com/Nexenta/nexentastor-docker-volume-plugin/issues/2) from Nexenta/review_comments
* Merge pull request [#3](https://github.com/Nexenta/nexentastor-docker-volume-plugin/issues/3) from Nexenta/build_issues

