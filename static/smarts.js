  var watchExampleVM = new Vue({
    el: '#app',
    data: {
      searchstring: "",
      books: [
        {"author": "Erwin", "title": "Awesome"}
      ]
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
      // In this case, we want to limit how often we access
      // yesno.wtf/api, waiting until the user has completely
      // finished typing before making the ajax request. To learn
      // more about the _.debounce function (and its cousin
      // _.throttle), visit: https://lodash.com/docs#debounce
      getBooks: _.debounce(
        function () {
          var vm = this
          axios.get('/books.json', {
            params: {
              filter: this.searchstring
            }
            
          })
            .then(function (response) {
              vm.books = response.data.books;
            })
            .catch(function (error) {
              vm.answer = 'Error! Could not reach the API. ' + error
            })
        },
        // This is the number of milliseconds we wait for the
        // user to stop typing.
        300
      )
    }
  })
