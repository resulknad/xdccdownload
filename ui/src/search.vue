<template>
    <div>
        <b-pagination :total-rows="pckgs.length" per-page="50" v-model="currentPage" class="my-0" />
          <b-table hover :items="pckgs" :fields="fields"              :current-page="currentPage"
             per-page="50">
          <template slot="actions" slot-scope="cell">
              <b-btn @click="showModal(cell.item)">Download</b-btn>
            </template></b-table>
              <b-modal @ok="download()" v-model="modalShow" title="Path">
                  {{configBaseDir}}
                 <b-form-input v-model="targetFolder"
                  type="text"
                  placeholder="Folder"></b-form-input>
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
        {
          key: 'Channel',
          sortable: true
        },
        {
          key: 'Filename',
          sortable: true
        },
        {
          key: 'Size',
          sortable: true
        }, "actions"],
        modalShow: false,
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
        showModal(p) {
            this.modalShow =true
            this.downloadId = p.ID
            this.configBaseDir = this.config.TargetPaths.filter(t => t.Type == p.Parsed.type)[0].Dir
            if (p.Parsed.type == "tvshow") {
                this.targetFolder = p.Parsed.title + "/" + "Season " + p.Parsed.season +"/"
            } else {
                this.targetFolder = p.Parsed.title + "/"

            }
        },
        loadData() {
            axios.get(consts.baseURL + `packages/` + encodeURIComponent(this.$route.query.search))
            .then(response => {
              this.pckgs = response.data
            })
            .catch(e => {
                console.log(e);
            });
            axios.get(consts.baseURL +  `config/`)
            .then(response => {
              this.config = response.data
            })
            .catch(e => {
                console.log(e);
            })
        },
        download() {
            axios.post(consts.baseURL + `download/`, {targetfolder: this.targetFolder, packid: this.downloadId})
            .then(response => {
                console.log(response)
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
