# BookBrowser
A tool to browse epubs and convert them to mobi ebooks using kindlegen. 


Heavily inspired by https://github.com/geek1011/BookBrowser/


## Features
- Search
- Advanced Search
    - Search any combination of fields
    - View all information in the results
- List view
- Responsive web interface
- Browse by:
    - Author
- Sorted by:
    - Last added
    - Alphabetically
- Conversion to mobi with Amazon [kindlegen](https://www.amazon.com/gp/feature.html?docId=1000765211)
- Easy-to-use
- Fast

## Requirements
kindlegen should be in $PATH or location provided with env var $KINDLEGEN_PATH

## Usage
Run BookBrowser from the directory with the epub books. By default, you can access the web interface at [http://localhost:8090](http://localhost:8090).

You can also use the command line arguments below:

````
  -addr string
    	The address to bind to. (default ":8090")
  -bookdir string
    	The directory to get books from. This directory must exist. (default ".")
  -tempdir string
    	The directory to use for storing temporary files such as book cover thumbnails. This directory is create on start and deleted on exit. (default is a subdirectory in your system's temp directory)
````
