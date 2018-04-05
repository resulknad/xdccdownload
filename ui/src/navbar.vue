<template>
    <b-navbar toggleable="md" type="dark" variant="info">

  <b-navbar-toggle target="nav_collapse"></b-navbar-toggle>

  <b-navbar-brand href="#">DL</b-navbar-brand>

  <b-collapse is-nav id="nav_collapse">

    <b-navbar-nav>
        <router-link to="tasks" tag="b-nav-item">

      <b-nav-item >Tasks</b-nav-item>
        </router-link>
        <router-link to="downloads" tag="b-nav-item">
      <b-nav-item >Downloads</b-nav-item>
        </router-link>
    </b-navbar-nav>

    <!-- Right aligned nav items -->
    <b-navbar-nav class="ml-auto">

      <b-nav-form>
        <b-form-input size="sm" v-model="search" class="mr-sm-2" type="text" placeholder="Search"/>
                    <router-link :to="{ name: 'search', query: { search }}" tag="b-button">
        <b-button size="sm" class="my-2 my-sm-0" type="submit">Search</b-button>
        </router-link>
      </b-nav-form>
    </b-navbar-nav>

  </b-collapse>
</b-navbar>
</template>

<script>
export default {
  name: 'dna_nav',
  data () {
    return {
      search: ''
    }
  }
}
</script>

<style>

</style>
