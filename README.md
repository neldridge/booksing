# booksing
A tool to browse epubs and convert them to mobi ebooks using kindlegen. 

Heavily inspired by https://github.com/geek1011/BookBrowser/


## Features
- Easy-to-use
- Searches for query in authors name and title of book
- List view
- "Responsive" web interface
- Sorted by Author
- Conversion to mobi with Amazon [kindlegen](https://www.amazon.com/gp/feature.html?docId=1000765211)
- automatic deletion of duplicates and unparsable epubs
  - epubs are marked as duplicates when the author and title are exactly the same (after fixing case and lastname, firstname issues)
  - if an epub is unparsable by booksing, it is deleted if ALLOW_DELETES is set, this doesn't nescecarily mean your ereader cannot read it!
- The first scan takes the longest, as all epubs are parsed from scratch, additional scans only parse new epubs.
  - setting ALLOW_DELETES speeds up this process as well, because duplicates get parsed every scan
- The speed is highly dependant on the DATABASE_LOCATION. If at all possible, place this on a SSD. This will speed up all operations a lot!
- With the DATABASE_LOCATION on an SSD, it is possible to re-scan more than 15.000 epubs on an external drive within a few seconds on limited (atom processor) hardware

## Requirements
- kindlegen should be in $PATH
- ebook-convert should be in $PATH (usually automatically installed when installing calibre)

## Usage
1. Run BookBrowser from the directory with the epub books. You can access the web interface at [http://localhost:7132](http://localhost:7132). 
1. Configure your smtp server, username and password. When using gmail / g suite, smtp server is smtp.gmail.com, username is your e-mail address and your regular account password (if you have 2FA enabled, please generate an application specific password) 
1. If you have a kindle, check the convert to mobi checkbox and enter your kindles email address (can be found on your amazon "devices" page) 
1. Optionally, edit the amount of results per page. The server has no real problem serving > 500 results per query, but the (mobile) browser usually has.
1. Press save when done.


You can use the following env vars to configure booksing:

````
  BOOK_DIR string
        The directory to get books from. This directory must exist. (default ".")
  ALLOW_DELETES bool
        Setting this to true makes booksing delete duplicates and unparsable (for booksing, your eReader may be more lenient) epubs from the filesystem *USE WITH CAUTION*
  DATABASE_LOCATION string
        Determines where to store the database (default: $BOOK_DIR/booksing.db)
````
