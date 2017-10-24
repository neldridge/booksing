# booksing
A tool to browse epubs and convert them to mobi ebooks using kindlegen. 

Heavily inspired by https://github.com/geek1011/BookBrowser/


## Features
- Easy-to-use
- Search
    - Search any combination of fields
- List view
- "Responsive" web interface
- Sorted by Author
- Conversion to mobi with Amazon [kindlegen](https://www.amazon.com/gp/feature.html?docId=1000765211)
- Pretty fast (can index over 10k epubs within a minute from an external hard drive on limited hardware, on my laptop it indexes about 300 epubs within a second)

## Requirements
- kindlegen should be in $PATH
- ebook-convert should be in $PATH (usually automatically installed when installing calibre)

## Usage
1. Run BookBrowser from the directory with the epub books. You can access the web interface at [http://localhost:7132](http://localhost:7132). 
1. Configure your smtp server, username and password. When using gmail / g suite, smtp server is smtp.gmail.com, username is your e-mail address and your regular account password (if you have 2FA enabled, please generate an application specific password) 
1. If you have a kindle, check the convert to mobi checkbox and enter your kindles email address (can be found on your amazon "devices" page) 
1. Optionally, edit the amount of results per page. The server has no real problem serving > 500 results per query, but the (mobile) browser usually has.
1. Press save when done.


You can use the following env var to configure where to find all the books:

````
  BOOK_DIR string
    	The directory to get books from. This directory must exist. (default ".")
````
