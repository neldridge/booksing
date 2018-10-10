<template>
<div class="container">
  <div class="box">

  <article class="media">
    <div class="media-content">
      <div class="content">
        <p>
          <strong>{{ book.title }}</strong>  <small>{{ book.author }}</small>
          <br>
          Lorem ipsum dolor sit amet, consectetur adipisckking elit. Aenean efficitur sit amet massa fringilla egestas. Nullam condimentum luctus turpis.
        </p>
      </div>
    </div>
  </article>

  </div>
</div>
</template>

<script>
import axios from "axios";
import lodash from "lodash";

export default {
  name: "BookInfo",
  props: ["hash"],
  data: function() {
    return {
      book: {}
    };
  },
  methods: {
    // _.debounce is a function provided by lodash to limit how
    // often a particularly expensive operation can be run.
    getBook: function() {
      var vm = this;
      axios
        .get("/book.json", {
          params: {
            hash: vm.hash
          }
        })
        .then(function(response) {
          vm.book = response.data;
        })
        .catch(function(error) {
          console.log(error);
        });
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
  },
  mounted: function() {
    this.getBook();
  }
};
</script>
