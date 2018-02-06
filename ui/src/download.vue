<template>
    <div>
          <b-table hover :items="pckgs" :fields="fields">
          <template slot="actions" slot-scope="cell">
              <b-btn size="sm" @click.stop="cancel(cell.item,cell.index,$event.target)">Stop</b-btn>
            </template></b-table>
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
        }, "actions"],
        intv: false
    }
  },
    methods: { 
        cancel(it) {
            axios.delete(consts.baseURL + `/download/` + it.Pack.ID)
            .then(response => {
                console.log(response)
            })
                
            .catch(e => {
                console.log(e);
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
