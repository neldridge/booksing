Vue.component('modal', {
  template: '#modal-template',
  props: ['show'],
  data: function() {
    return {
      email: localStorage.getItem("email"),
      smtpserver: localStorage.getItem("smtpserver"),
      smtpuser: localStorage.getItem("smtpuser"),
      smtppass: localStorage.getItem("smtppass"),
      convert: localStorage.getItem("convert")
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
      
      this.close();
    }
  }
});

  var watchExampleVM = new Vue({
    el: '#app',
    data: {
      searchstring: "",
      books: [ ],
      showModal: false,
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
          vm.statusMessage = "getting results"
          axios.get('/books.json', {
            params: {
              filter: this.searchstring
            }
            
          })
            .then(function (response) {
              vm.books = response.data.books;
              vm.searchDone = true;
            })
            .catch(function (error) {
              vm.answer = 'Error! Could not reach the API. ' + error
            })
        },
        // This is the number of milliseconds we wait for the
        // user to stop typing.
        100
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
      sendBookToKindle: function (bookid) {
        axios.post('/convert', {
            bookid: bookid,
            email: "gnur@free.kindle.com"
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
