# booksing
A tool to browse epubs.

Heavily inspired by https://github.com/geek1011/BookBrowser/

## Installation
Download an appropriate release from the [release](https://github.com/gnur/booksing/releases) page


## Features
- Easy-to-use
- Google sign in
- Super fast search thanks to meilisearch (can easily search 100k+ books within 500ms)
- QR based login flow for lower powered devices (use your phone to grant access to your ereader)
- List view
- "Responsive" web interface
- Automatic deletion of duplicates and unparsable epubs
- Create bookmarks to track what you want to read, have read and stopped reading
- Grant access from the admin page
- See what books have been downloaded

## Requirements
Booksing only depends on [meilisearch](https://www.meilisearch.com/).

## Configuration

Set the following env vars to configure booksing:

| env var               | default                 | required           | purpose                                                                                                         |
|-----------------------|-------------------------|--------------------|-----------------------------------------------------------------------------------------------------------------|
| BOOKSING_ADMINUSER    | `-`                     | :white_check_mark: | This determines the admin user, the only user that can login by default                                         |
| BOOKSING_BOOKDIR      | `.`                     | :x:                | The directery where books are stored after importing                                                            |
| BOOKSING_FAILDIR      | `./failed`              | :x:                | The directory where books are moved if the import fails                                                         |
| BOOKSING_FQDN         | `http://localhost:7132` | :x:                | used to set the session cookie and callback for google auth                                                     |
| BOOKSING_IMPORTDIR    | `./import`              | :x:                | The directory where booksing will periodically look for books                                                   |
| BOOKSING_LOGLEVEL     | `info`                  | :x:                | determines the loglevel, supported values: error, warning, info, debug                                          |
| BOOKSING_MEILI_HOST   | `http://localhost:7700` | :x:                | [meilisearch](https://www.meilisearch.com/) host (used for storing book information)                            |
| BOOKSING_MEILI_INDEX  | `books`                 | :x:                | The index used in meilisearch to store the book information                                                     |
| BOOKSING_MEILI_KEY    | `-`                     | :white_check_mark: | The key used to put stuff in meilisearch, needs write access                                                    |
| BOOKSING_BATCHSIZE    | `50`                    | :x:                | The amount of books that will be stored in the databases at a time                                              |
| BOOKSING_BINDADDRESS  | `localhost:7132`        | :x:                | The bind address, if external access is needed this should be changed to `:7132`                                |
| BOOKSING_DATABASE     | `file://booksing.db`    | :x:                | The path to put the database file (bbolt based)                                                                 |
| BOOKSING_SAVEINTERVAL | `10s`                   | :x:                | The time between saves if the batchsize is not reached yet                                                      |
| BOOKSING_SECRET       | `-`                     | :white_check_mark: | The secret used to encrypt the session cookie                                                                   |
| BOOKSING_TIMEZONE     | `Europe/Amsterdam`      | :x:                | Timezone used for storing all time information                                                                  |
| BOOKSING_WORKERS      | `5`                     | :x:                | Amount of parallel workers used for parsing epubs                                                               |
| GIN_MODE              | `-`                     | :x:                | Set to `release` to make gin (the request router) less verbose and faster                                       |
| GOOGLE_KEY            | `-`                     | :white_check_mark: | google key (see https://developers.google.com/identity/protocols/oauth2/openid-connect to see how to get these) |
| GOOGLE_SECRET         | `-`                     | :white_check_mark: | See above                                                                                                       |




## Usage
1. Run the booksing binary from the directory with the epub books. You can access the web interface at [http://localhost:7132](http://localhost:7132) 
