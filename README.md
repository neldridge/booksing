# BookBrowser
A tool to browse epubs and convert them to mobi ebooks using kindlegen. 


Heavily inspired by https://github.com/geek1011/BookBrowser/


## Features
- Easy-to-use
- Search
    - Search any combination of fields
- List view
- Responsive web interface
- Sorted by Author
- Conversion to mobi with Amazon [kindlegen](https://www.amazon.com/gp/feature.html?docId=1000765211)
- Fast (after initial loading of epubs)

## Requirements
kindlegen should be in $PATH or location provided with env var $KINDLEGEN_PATH

## Usage
Run BookBrowser from the directory with the epub books. By default, you can access the web interface at [http://localhost:7132](http://localhost:7132).

You can use the following env var to configure where to find all the books:

````
  BOOK_DIR string
    	The directory to get books from. This directory must exist. (default ".")
````
