# Changelog

All notable changes to `webklex/tbm` will be documented in this file.

Updates should follow the [Keep a CHANGELOG](http://keepachangelog.com/) principles.


## [UNRELEASED]
### Fixed
- NaN

### Added
- NaN

### Breaking changes
- NaN


## [2.1.0] - 2022-12-13
### Fixed
- Newly required twitter api attributes added

### Added
- Pagination support added
- Search extended
- Programmatic and safe program interruption added #19
- Status page added


## [2.0.0] - 2022-11-21
### Fixed
- Prevent multiple parallel runners
- Changed config file location to program basedir #14 (thanks @Wikinaut)
- Increase bookmark cursor if the count limit is reached
- Mixed bookmark response type handling

### Added
- Offline mode support added (load everything from local sources)
- Thread view added
- (Optional) Remove bookmarked tweets after download using `--danger-remove-bookmarks`
- Output colors and additional error outputs added
- Show a skip message for already downloaded tweets #15 (thanks @Wikinaut)
- Fetch the access token and all required sections automatically

### Breaking changes
- Default config location has changed to `config.json`


## [1.0.1] - 2022-11-19
### Fixed
- Missing tweet index added
- Additional api error handling added
- Alternative request features added (support newer queries)


## [1.0.0] - 2022-11-19
### Fixed
- Identical Hashtag and Reference linking fixed

### Added
- Video download support added
- Default config added
- Original source back-links
- Result counter added to the gui

### Breaking changes
- Previously cached json files aren't supported


## [0.1.0] - 2022-11-11
### Added
- Local media cache added


## [0.0.1] - 2022-08-10
Initial release
