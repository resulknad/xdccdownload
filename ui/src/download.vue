<template>
    <div>
          <b-table hover :items="pckgs" :fields="fields">
          <template slot="actions" slot-scope="cell">


            </template>          <template slot="actions" slot-scope="cell">
              <b-btn @click="showModal(cell.item.Messages)">Info</b-btn>
              <b-btn size="sm" @click.stop="retry(cell.item,cell.index,$event.target)">Retry</b-btn>
              <b-btn size="sm" @click.stop="cancel(cell.item,cell.index,$event.target)">Stop</b-btn>
            </template></b-table>
              </b-table>
              
              <b-modal v-model="modalShow" title="Info">
                  {{info}}
                  <p class="my-4">{{downloadId}}</p>
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
        pckgs: [],
          fields: [
              {key: 'Pack', formatter: (s) => s.Filename},

        {
          key: 'Percentage',
          sortable: true
        }, 
              {key:'Messages'},
"actions"],
        intv: false,
        modalShow: false,
        info: ""
    }
  },
    methods: { 
        showModal(msgs) {
            this.modalShow = true
            this.info = msgs
        },
        cancel(it) {
            axios.delete(consts.baseURL + `download/` + it.ID)
            .then(response => {
                console.log(response)
            })
                
            .catch(e => {
                console.log(e);
            })
        },
        retry(p) {
            axios.post(consts.baseURL + `download/`, {targetfolder: p.Targetfolder, packid: p.Pack.ID})
            .then(response => {
                console.log(response)
            })

        },
        loadData: function() {
    axios.get(consts.baseURL + `download/`)
    .then(response => {
      this.pckgs = response.data
    })
    .catch(e => {
        console.log(e);
    })
        }},
    beforeDestroy() {
    clearInterval(this.intv);
    },
  mounted() {
      if (this.intv == false) {
          this.intv = setInterval(function () {
      this.loadData();
    }.bind(this), 500);
      }
    this.loadData()
  }

}
</script>

<style>

</style>
