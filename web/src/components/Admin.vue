<template>
  <section>
    <router-link :to="{ name: 'home' }" class="button field is-info is-medium">back to search</router-link>
    <article v-if="showWarning" class="message is-warning">
      <div class="message-header">
        <p>{{ warningMessage }}</p>
      </div>
    </article>
    <article v-if="showSuccess" class="message is-success">
      <div class="message-header">
        <p>{{ successMessage }}</p>
      </div>
    </article>
    <b-tabs position="is-centered" class="block">
      <b-tab-item label="downloads">
        <b-table :data="downloads" paginated striped narrowed per-page="50">
          <template slot-scope="props">
            <b-table-column
              field="timestamp"
              label="timestamp"
            >{{ formatDate(props.row.timestamp) }}</b-table-column>
            <b-table-column field="user" label="user">{{ props.row.user }}</b-table-column>
            <b-table-column field="book" label="book">{{ props.row.hash }}</b-table-column>
          </template>
        </b-table>
      </b-tab-item>
      <b-tab-item label="api keys">
        <b-table :data="apikeys">
          <template slot-scope="props">
            <b-table-column field="id" label="id">
              {{
              props.row.id
              }}
            </b-table-column>
            <b-table-column field="created" label="created">
              {{
              formatDate(props.row.Created)
              }}
            </b-table-column>
            <b-table-column field="last use" label="last use">
              {{
              formatDate(props.row.LastUsed)
              }}
            </b-table-column>
            <b-table-column field="id" label="delete">
              <a @click="confirmDelete(props.row.Key)">
                <span class="icon">
                  <i class="mdi mdi-delete"></i>
                </span>
              </a>
            </b-table-column>
          </template>
        </b-table>
        <!-- add api key code -->
        <div class="field has-addons">
          <div class="control">
            <input v-model="apikeyid" class="input" type="text" placeholder="API user" />
          </div>
          <div class="control">
            <a class="button is-info" @click="addAPIKey">save</a>
          </div>
        </div>
        <!-- end api key code -->
      </b-tab-item>
      <b-tab-item label="users">
        <b-table :data="users">
          <template slot-scope="props">
            <b-table-column field="id" label="id">
              {{
              props.row.Username
              }}
            </b-table-column>
            <b-table-column field="created" label="created">
              {{
              formatDate(props.row.Created)
              }}
            </b-table-column>
            <b-table-column field="last seen" label="last seen">
              {{
              formatDate(props.row.LastSeen)
              }}
            </b-table-column>
            <b-table-column field="IsAllowed" label="has access">
              <a @click="toggleAccess(props.row)">
                {{
                props.row.IsAllowed
                }}
              </a>
            </b-table-column>
            <b-table-column field="IsAdmin" label="admin">
              {{
              props.row.IsAdmin
              }}
            </b-table-column>
          </template>
        </b-table>
      </b-tab-item>
    </b-tabs>
  </section>
</template>

<script>
import axios from "axios";
import lodash from "lodash";

export default {
  name: "home",
  data: function() {
    return {
      downloads: [],
      isAdmin: false,
      apikeys: [],
      users: [],
      apikeyid: "",
      showWarning: false,
      showSuccess: false,
      warningMessage: "",
      successMessage: "",
      username: this.$store.getters.username
    };
  },
  mounted: function() {
    this.getDownloads();
    this.getAPIKeys();
    this.getUsers();
    this.getUser();
  },

  methods: {
    formatDate(dateStr) {
      var d = new Date(dateStr);
      var input = new Date(dateStr);
      var today = new Date();
      if (d.setHours(0, 0, 0, 0) == today.setHours(0, 0, 0, 0)) {
        return input.toLocaleTimeString("nl-NL", {});
      } else {
        return input.toLocaleString("nl-NL", {});
      }
    },
    getUser: function() {
      var vm = this;
      axios
        .get("/auth/user.json")
        .then(function(response) {
          vm.isAdmin = response.data.admin;
        })
        .catch(function(error) {
          console.log(error);
        });
    },
    getUsers: function() {
      var vm = this;
      axios
        .get("/admin/users")
        .then(function(response) {
          vm.users = response.data;
        })
        .catch(function(error) {
          console.log(error);
        });
    },
    formatDateTime(dateStr) {
      return dateStr.replace("T", " ").substr(0, 16);
    },
    getDuration(startDateStr, endDateStr) {
      var date1 = new Date(startDateStr);
      var date2 = new Date(endDateStr);

      var msec = date2.getTime() - date1.getTime();

      var returnstring = "";
      var hh = Math.floor(msec / 1000 / 60 / 60);
      if (hh > 0) {
        returnstring += h + "h";
      }
      msec -= hh * 1000 * 60 * 60;
      var mm = Math.floor(msec / 1000 / 60);
      if (mm > 0) {
        returnstring += mm + "m";
      }
      msec -= mm * 1000 * 60;
      var ss = Math.floor(msec / 1000);
      if (ss > 0) {
        returnstring += ss + "s";
      }
      msec -= ss * 1000;
      if (ss == 0 && msec > 0) {
        returnstring += msec + "ms";
      }

      return returnstring;
    },
    getAPIKeys: function() {
      axios
        .get("/auth/apikey")
        .then(resp => {
          if (resp.data.user.APIKeys != null) {
            this.apikeys = resp.data.user.APIKeys;
          } else {
            this.apikeys = [];
          }
        })
        .catch(function(error) {
          console.log(error);
        });
    },
    confirmDelete: function(uuid) {
      this.$dialog.confirm({
        message: "Delete this api key",
        type: "is-danger",
        hasIcon: true,
        onConfirm: () => this.deleteAPIKey(uuid)
      });
    },
    deleteAPIKey: function(uuid) {
      axios.delete("/auth/apikey/" + encodeURIComponent(uuid)).then(
        () => {
          this.refresh();
          this.$toast.open({
            duration: 2000,
            type: "is-success",
            message: "key deleted",
            position: "is-bottom"
          });
        },
        err => {
          this.$toast.open({
            duration: 3000,
            type: "is-danger",
            message: "note failed to delete: " + err,
            position: "is-bottom"
          });
          console.log(err);
        }
      );
    },
    addAPIKey() {
      var vm = this;
      fetch("/auth/apikey", {
        method: "POST",
        headers: {
          "Content-Type": "application/json"
        },
        body: JSON.stringify({
          id: this.apikeyid
        })
      })
        .then(resp => resp.json())
        .then(resp => {
          this.apikeyid = "";
          this.showSuccessMsg("api key added: " + resp.key.Key);
          this.getAPIKeys();
        })
        .catch(err => {
          this.showErrorAlert("failed adding api key: " + err);
        });
    },
    toggleAccess: function(user) {
      var vm = this;
      axios
        .post("/admin/user/" + user.Username, {
          IsAllowed: !user.IsAllowed
        })
        .then(resp => {
          this.getUsers();
        })
        .catch(err => {
          this.showErrorAlert("failed adding api key: " + err);
        });
    },
    showSuccessMsg: function(msg) {
      this.successMessage = msg;
      this.showSuccess = true;
    },

    showErrorAlert: function(msg) {
      this.warningMessage = msg;
      this.showWarning = true;
    },

    hideErrorAlert: function() {
      this.showWarning = false;
    },
    getDownloads: function() {
      var vm = this;
      axios
        .get("/admin/downloads.json")
        .then(function(response) {
          if (response.data !== null) {
            vm.downloads = response.data;
          } else {
            vm.downloads = [];
          }
        })
        .catch(function(error) {
          console.log(error);
        });
    }
  }
};
</script>
