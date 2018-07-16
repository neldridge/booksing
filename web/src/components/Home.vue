<template>
<div id="app" class="container">

<nav class="level">
    <form action="javascript:void()" onsubmit="return false">
      <div class="level-item">
      <div class="field has-addons">
        <p class="control">
        <input class="input" v-model="searchstring" id="search" placeholder="type here to search" type="text">
        </p>
        <p class="control">
          <button class="button">
            Search
          </button>
        </p>
      </div>
      </div>
    </form>
    <!--
      <div class="level-item">
        <a class="button" v-on:click.stop="toggleSettings">settings</a>
      </div>
      -->
    </nav>
  
  <div class="modal" id="settings">
    <div class="modal-background"></div>
    <div class="modal-content">
      <div class="card">
        <div class="card-content">
          <p class="subtitle"> werkt t beter</p>
          <p class="subtitle"> met echte content?</p>
        </div>
        <footer class="card-footer">
          <a class="card-footer-item" v-on:click.stop="refreshBooklist">
            {{ refreshButtonText }}
          </a>
          <a class="card-footer-item">
            save
          </a>
        </footer>
      </div>
    </div>
    <button class="modal-close is-large" aria-label="close" v-on:click.stop="toggleSettings"></button>
  </div>

  <div class="section">
        <table class="table is-striped is-narrow is-hoverable is-fullwidth">
          <thead>
            <tr>
              <th>author</th>
              <th>title</th>
              <th></th>
            </tr>
          </thead>
          <tbody>
            <template v-for="book in books">
            <tr v-bind:key="book.hash" v-on:click.stop="toggleModal(book.hash)">
              <td>{{ book.author }}</td>
              <td>{{ book.title }}</td>
              <td>
               <div class="modal" :id="book.hash">
                 <div class="modal-background"></div>
                 <div class="modal-content">
                   <!-- Any other Bulma elements you want -->
                   <div class="card">
                 <div class="card-content">
                   <p class="subtitle">
                     {{ book.author }}
                   </p>
                   <p class="subtitle">
                     {{ book.title }}
                   </p>
                   {{ book.description }}
                 </div>
                 <footer class="card-footer">
                   <a class="card-footer-item" :href="'/download/?book=' + book.filename">
                       .epub
                   </a>
                   <a class="card-footer-item" v-if="book.hasmobi" :href="'/download/?book=' + book.filename.replace('epub', 'mobi')">
                       .mobi
                   </a>
                   <a class="card-footer-item" v-else v-on:click.stop="convertBook(book.hash)">
                     <template v-if="converting">
                       <a class="button is-info is-loading">
                         converting
                       </a>
                     </template>
                     <template v-else>
                       convert
                     </template>
                   </a>
                   <a class="card-footer-item" v-on:click.stop="deleteBook(book.hash)">
                       delete
                   </a>
                 </footer>
               </div>
                 </div>
                 <button class="modal-close is-large" aria-label="close" v-on:click.stop="toggleModal(book.hash)"></button>
               </div>
              </td>
            </tr>
            </template>
          </tbody>
        </table>
     
  </div>
</div>
</template>

<script>
import axios from "axios";
import lodash from "lodash";

export default {
  name: "home",
  data: function() {
    return {
      searchstring: "",
      books: [],
      total: 0,
      enableSend: localStorage.getItem("enablesend") === "true",
      descriptionVisible: false,
      description: "",
      email: "test",
      converting: false,
      searchDone: false,
      statusMessage: "please enter your query",
      refreshButtonText: "refresh"
    };
  },
  watch: {
    // whenever question changes, this function will run
    searchstring: function(newQuestion) {
      this.getBooks();
    }
  },
  mounted: function() {
    this.getBooks();
  },

  methods: {
    convertBook: function(hash) {
      console.log(hash);
      var vm = this;
      this.converting = true;
      const params = new URLSearchParams();
      params.append('hash', hash);
        axios
          .post("/convert/", params)
          .then(function(response) {
            vm.getBooks();
            console.log(response);
            vm.converting = false;
          })
          .catch(function(error) {
            vm.converting = false;
            console.log(error);
          });

    },
    deleteBook: function(hash) {
      var vm = this;
      const params = new URLSearchParams();
      params.append('hash', hash);
        axios
          .post("/delete/", params)
          .then(function(response) {
            vm.toggleModal(hash);
            vm.getBooks();
          })
          .catch(function(error) {
            vm.converting = false;
            console.log(error);
          });

    },
    toggleSettings: function(hash) {
      var modal = document.getElementById("settings");
      modal.classList.toggle("is-active");
    },
    toggleModal: function(hash) {
      var modal = document.getElementById(hash);
      modal.classList.toggle("is-active");
      console.log(hash);
    },
    getBooks: lodash.debounce(
      function() {
        var vm = this;
        vm.searchDone = false;
        vm.statusMessage = "getting results";
        axios
          .get("/books.json", {
            params: {
              filter: this.searchstring,
              results: 100
            }
          })
          .then(function(response) {
            vm.books = response.data.books;
            vm.total = response.data.total;
            document.title = `booksing - ${vm.total} books available for searching`
            vm.searchDone = true;
          })
          .catch(function(error) {
            vm.statusMessage = "Something went wrong";
            console.log(error);
          });
      },
      // This is the number of milliseconds we wait for the
      // user to stop typing.
      500
    ),
    refreshBooklist: function() {
      var vm = this;
      vm.refreshButtonText = "Refreshing...";
      axios
        .get("/refresh")
        .then(function(response) {
          vm.refreshButtonText = "refresh";
          vm.getBooks();
          var modal = document.getElementById("settings");
          modal.classList.remove("is-active");
        })
        .catch(function(error) {
          vm.refreshButtonText = "refresh";
          console.log(error);
        });
    },
    showDescription: function(book) {
      var vm = this;
      vm.description = book.description;
      vm.descriptionVisible = true;
    },
    sendBookToKindle: function(id) {
      axios
        .post("/convert", {
          bookhash: id,
          email: localStorage.getItem("email"),
          smtpserver: localStorage.getItem("smtpserver"),
          smtpuser: localStorage.getItem("smtpuser"),
          smtppass: localStorage.getItem("smtppass"),
          convert: localStorage.getItem("convert") === "true"
        })
        .then(function(response) {
          console.log(response);
        })
        .catch(function(error) {
          console.log(error);
        });
    }
  }
};
</script>
