# booksing
<img src="./gopher.png" width="350" alt="nerdy gopher">
A tool to browse epubs.

Kind of inspired by https://github.com/geek1011/BookBrowser/

## Installation
Download an appropriate release from the [release](https://github.com/gnur/booksing/releases) page


## Features
- Easy-to-use
- List view
- "Responsive" web interface
- Automatic deletion of duplicates and unparsable epubs
- Automatic sorting of books based on Author
- See what books have been downloaded
- If you have an authenticating proxy booksing can determine the username from a header, and the admin user will be able to grant users access.

## todo
- make sure sqlite features work reliably
- fix goreleaser so it can build with sqlite for all platforms

## Requirements
- none

## Configuration

Set the following env vars to configure booksing:

| env var               | default                | required           | purpose                                                                                                                  |
|-----------------------|------------------------|--------------------|--------------------------------------------------------------------------------------------------------------------------|
| BOOKSING_ADMINUSER    | `unknown`              | :x:                | This determines the admin user, the only user that can login by default unless `allowallusers` is set                    |
| BOOKSING_ALLOWALLUSER | `true`                 | :x:                | This determines whether all users can login                                                                              |
| BOOKSING_BATCHSIZE    | `50`                   | :x:                | The amount of books that will be stored in the databases at a time                                                       |
| BOOKSING_BINDADDRESS  | `localhost:7132`       | :x:                | The bind address, if external access is needed this should be changed to `:7132`                                         |
| BOOKSING_BOOKDIR      | `.`                    | :x:                | The directory where books are stored after importing                                                                     |
| BOOKSING_DATABASEDIR  | `./db/`                | :x:                | The path to put the database files (sqlite based)                                                                        |
| BOOKSING_FAILDIR      | `./failed`             | :x:                | The directory where books are moved if the import fails                                                                  |
| BOOKSING_IMPORTDIR    | `./import`             | :x:                | The directory where booksing will periodically look for books                                                            |
| BOOKSING_LOGLEVEL     | `info`                 | :x:                | determines the loglevel, supported values: error, warning, info, debug                                                   |
| BOOKSING_TIMEZONE     | `Europe/Amsterdam`     | :x:                | Timezone used for storing all time information                                                                           |
| BOOKSING_USERHEADER   | `-`                    | :x:                | The header to take the username from (if behind cloudflare access, this should be: `Cf-Access-Authenticated-User-Email`) |


## Tips
- For large collections, it is perfectly acceptable to place the ebooks themselves on an external USB drive, but you should place the database dir on a faster (preferable SSD) disk.

## Example first run

```

$ mkdir booksing booksing/failed booksing/import booksing/db 
$ cd booksing
$ wget 'https://github.com/gnur/booksing/releases/download/v8.0.1/booksing_8.0.1_linux_x86_64.tar.gz'
$ tar xzf booksing*
$ ./booksing &
$ mv ~/library/*.epub import/
# visit localhost:7132 to see the books in the interface
