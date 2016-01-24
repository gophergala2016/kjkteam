Imagine you have uncommited changes in your git repository and want to
preview them before commiting.

Type `differ` and get browser-based, efficient UI for previewing the changes. Inception shot:

![Differ Screenshot](differ.png)

Use `j`/`k` for next/previous file.

Differ is port of https://github.com/danvk/webdiff to Go.

Mac binary: [differ](https://dl.dropboxusercontent.com/u/3064436/differ)

To build from sources:
* `go get -u github.com/gophergala2016/kjkteam` or `git clone https://github.com/gophergala2016/kjkteam.git`
* must have node installed
* `npm install` to get the needed JavaScript libraries for the front-end
* `scripts/build.sh` to build a self-contained `differ` executable

Not good enought to win? I beg to differ.

TODO:
* directory compare
* `git scdiff` support
* refresh the diff on / reload and on "focus" event on window
* support for images
* -share option that sends data to central server for sharing with other people
* native mac app
* native windows app
