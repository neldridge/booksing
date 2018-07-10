Vue.component('modal', {
  template: '#modal-template',
  props: ['show'],
  data: function() {
    return {
      email: localStorage.getItem("email"),
      smtpserver: localStorage.getItem("smtpserver"),
      smtpuser: localStorage.getItem("smtpuser"),
      smtppass: localStorage.getItem("smtppass"),
      convert: localStorage.getItem("convert") == "true",
      enableSend: localStorage.getItem("enablesend") =="true",
      resultcount: 50
    }
  },
  methods: {
    close: function () {
      this.$emit('close');
    },
    savePost: function () {
      // Some save logic goes here...
      localStorage.setItem("email", this.email)
      localStorage.setItem("smtpserver", this.smtpserver)
      localStorage.setItem("smtpuser", this.smtpuser)
      localStorage.setItem("smtppass", this.smtppass)
      localStorage.setItem("convert", this.convert)
      localStorage.setItem("enablesend", this.enableSend)
      localStorage.setItem("resultcount", this.resultcount)
      this.close();
    }
}
});

  var watchExampleVM = new Vue({
    el: '#app',
    data: {
      searchstring: "",
      books: [ ],
      total: 0,
      enableSend: localStorage.getItem("enablesend") == "true",
      showModal: false,
      descriptionVisible: false,
      description: "",
      email: "test",
      searchDone: false,
      statusMessage: "please enter your query",
      refreshButtonText: "refresh"
    },
    watch: {
      // whenever question changes, this function will run
      searchstring: function (newQuestion) {
        this.getBooks()
      }
    },

    methods: {
      // _.debounce is a function provided by lodash to limit how
      // often a particularly expensive operation can be run.
      getBooks: _.debounce(
        function () {
          var vm = this
          vm.searchDone = false;
          vm.statusMessage = "getting results"
          axios.get('/books.json', {
            params: {
              filter: this.searchstring,
              results: localStorage.getItem("resultcount")
            }
            
          })
            .then(function (response) {
              vm.books = response.data.books;
              vm.total = response.data.total;
              vm.searchDone = true;
            })
            .catch(function (error) {
              vm.statusMessage = "Something went wrong"
            })
        },
        // This is the number of milliseconds we wait for the
        // user to stop typing.
        500
      ),
      refreshBooklist: function() {
        var vm = this
        vm.refreshButtonText = "Refreshing..."
        axios.get('/refresh')
            .then(function (response) {
              vm.refreshButtonText = "refresh"
              vm.getBooks()
            })
            .catch(function (error) {
              vm.refreshButtonText = "refresh"
            })

      },
      showDescription: function (book) {
        var vm = this;
        vm.description = book.description;
        vm.descriptionVisible = true;
      },
      sendBookToKindle: function (id) {
        axios.post('/convert', {
            bookhash: id,
            email: localStorage.getItem("email"),
            smtpserver: localStorage.getItem("smtpserver"),
            smtpuser: localStorage.getItem("smtpuser"),
            smtppass: localStorage.getItem("smtppass"),
            convert: (localStorage.getItem("convert") == "true"),
          })
          .then(function (response) {
            console.log(response)
          })
          .catch(function (error) {
            console.log(error)
          })
        console.log(event)
      }
   }
})
