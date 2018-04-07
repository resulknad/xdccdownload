<template>
    <div>

        <b-table hover :items="tasks" :fields="fields">
          <template slot="actions" slot-scope="cell">
              <b-btn @click="edit(cell.item)">Edit</b-btn>
              <b-btn @click="del(cell.item)">Del</b-btn>
            </template>
        </b-table>

        <b-button size="sm" class="my-2 my-sm-0" @click="createTask()" type="submit">Add</b-button>
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
        createTask() {
            axios.post(consts.baseURL + `tasks/`, {"Name": "new task"})
            .then(response => {
              this.tasks = response.data
            })
            .catch(e => {
                console.log(e);
            });
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
        del(task) {
            axios.delete(consts.baseURL + `tasks/` + task.ID)
            .then(response => {
                console.log(response)
                this.loadData()
            })
                
            .catch(e => {
                this.loadData()
            })
        },
        save() {
            this.showEditModal = true
            axios.put(consts.baseURL + `tasks/` + this.el.ID, this.el)
            .then(response => {
                console.log(response)
                this.showEditModal = false
                this.loadData()
            })
                
            .catch(e => {
                this.showEditModal = true
                console.log(e);
            })
        }
    }
}
</script>

<style>

</style>
