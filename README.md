# booksing
<img src="./gopher.png" width="350" alt="nerdy gopher">
A tool to browse epubs.

Heavily inspired by https://github.com/geek1011/BookBrowser/

## Installation
Download an appropriate release from the [release](https://github.com/gnur/booksing/releases) page


## Features
- Easy-to-use
- List view
- "Responsive" web interface
- Automatic deletion of duplicates and unparsable epubs
- Create bookmarks to track what you want to read, have read and stopped reading
- Grant access from the admin page
- See what books have been downloaded

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
| BOOKSING_BOOKDIR      | `.`                    | :x:                | The directery where books are stored after importing                                                                     |
| BOOKSING_DATABASEDIR  | `./db/`                | :x:                | The path to put the database files (bbolt based)                                                                         |
| BOOKSING_FAILDIR      | `./failed`             | :x:                | The directory where books are moved if the import fails                                                                  |
| BOOKSING_IMPORTDIR    | `./import`             | :x:                | The directory where booksing will periodically look for books                                                            |
| BOOKSING_LOGLEVEL     | `info`                 | :x:                | determines the loglevel, supported values: error, warning, info, debug                                                   |
| BOOKSING_MQTTCLIENTID | `booksing`             | :x:                | Default client ID used in MQTT events                                                                                    |
| BOOKSING_MQTTENABLE   | `false`                | :x:                | This determines if booksing will send out certain "events" on MQTT                                                       |
| BOOKSING_MQTTHOST     | `tcp://localhost:1883` | :x:                | The host to send events to                                                                                               |
| BOOKSING_MQTTTOPIC    | `events`               | :x:                | The topic prefix to push events to                                                                                       |
| BOOKSING_SAVEINTERVAL | `10s`                  | :x:                | The time between saves if the batchsize is not reached yet                                                               |
| BOOKSING_TIMEZONE     | `Europe/Amsterdam`     | :x:                | Timezone used for storing all time information                                                                           |
| BOOKSING_USERHEADER   | `-`                    | :x:                | The header to take the username from (if behind cloudflare access, this should be: `Cf-Access-Authenticated-User-Email`) |
| BOOKSING_WORKERS      | `5`                    | :x:                | Amount of parallel workers used for parsing epubs                                                                        |



## Example first run

```

$ mkdir booksing booksing/failed booksing/import booksing/db 
$ cd booksing
$ wget 'https://github.com/gnur/booksing/releases/download/v8.0.1/booksing_8.0.1_linux_x86_64.tar.gz'
$ tar xzf booksing*
$ ./booksing &
$ mv ~/library/*.epub import/
# visit localhost:7132 to see the books in the interface
