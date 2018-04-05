<template>
    <div>
        <b-table hover :items="tasks" :fields="fields">
          <template slot="actions" slot-scope="cell">
              <b-btn @click="edit(cell.item)">Edit</b-btn>
            </template>
        </b-table>

              <b-modal @ok="save()" v-model="showEditModal" title="">
                  <b-form-input v-model="el.Name" type="text" placeholder="Name"></b-form-input>
                  <b-form-input v-model="el.Criteria" type="text" placeholder="Criteria"></b-form-input>
                      <br>
                      <div>
                          {{el.Enabled}}
                      <input type="checkbox" v-model="el.Enabled">
                      </input>
                      </div>
              </b-modal>

    </div>

</template>

<script>
import axios from 'axios';
import consts from './const.js'
export default {
  name: 'search',

  data () {
    return {

        status: false,
        showEditModal: false,
        el: {Enabled: false},
        tasks: [],
          fields: [
        {
          key: 'Name',
          sortable: true
        },
        {
          key: 'Criteria',
          sortable: true
        },
        {
          key: 'State',
          sortable: true
        }, "actions"],
        showEditModal: false,
        configBaseDir: '',
        downloadId: -1,
        currentPage: 0
    }
  },

  created() {
    this.loadData()
  },
watch: {
    '$route.query.search' (to, from) {
        this.loadData()
    }
},
    
    methods: {
        edit(task) {
            this.el = task
            this.showEditModal = true
        },
        loadData() {
            axios.get(consts.baseURL + `tasks/`)
            .then(response => {
              this.tasks = response.data
            })
            .catch(e => {
                console.log(e);
            });
        },
        save() {
            axios.put(consts.baseURL + `tasks/` + this.el.ID, this.el)
            .then(response => {
                console.log(response)
                this.loadData()
            })
                
            .catch(e => {
                console.log(e);
            })
        }
    }
}
</script>

<style>

</style>
