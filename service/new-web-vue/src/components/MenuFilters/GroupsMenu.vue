<template>
  <a
    href="#"
    class="navbar-filter__link"
    onclick="return false"
    @click="toggleMenu"
  >
    <font-awesome-icon
      class="fa-fw"
      icon="users"
      v-if="!current_group || !$route.params.id"
    />
    <font-awesome-icon
      class="fa-fw"
      icon="user"
      v-else-if="current_group && $route.params.id == 10"
    />
    <img
      draggable="false"
      :src="current_group.GroupIcon"
      :alt="current_group.GroupName"
      class="navbar-filter__img"
      v-else-if="current_group && $route.params.id && $route.params.id != 10"
    />
    <span class="navbar-filter__span" v-if="!current_group || !$route.params.id"
      >Groups</span
    >
    <span class="navbar-filter__span" v-else>{{
      (
        current_group.GroupName.charAt(0).toUpperCase() +
        current_group.GroupName.slice(1)
      ).replace("_", " ")
    }}</span>
  </a>
  <ul class="navbar-filter-items peer-focus-within:scale-y-100">
    <li class="navbar-pending" v-if="groups.length < 1">
      <img
        draggable="false"
        :src="`/assets/loading/${Math.floor(Math.random() * 7)}.gif`"
        class="navbar-pending__img"
      />
    </li>
    <li class="navbar-filter-item" v-for="group in groups" :key="group.ID">
      <router-link
        :to="`/vtubers/${group.ID || ''}`"
        class="navbar-filter-item__link"
      >
        <font-awesome-icon
          class="fa-fw navbar-filter-item__svg"
          icon="users"
          v-if="!group.ID"
        />
        <font-awesome-icon
          class="fa-fw navbar-filter-item__svg"
          icon="user"
          v-else-if="group.ID && group.ID == 10"
        />
        <img
          draggable="false"
          v-else-if="group.ID && group.ID != 10"
          :src="group.GroupIcon"
          :alt="group.GroupName"
          class="navbar-filter-item__img"
        />
        <span class="navbar-filter-item__span">
          {{
            (
              group.GroupName.charAt(0).toUpperCase() + group.GroupName.slice(1)
            ).replace("_", " ")
          }}
        </span>
      </router-link>
    </li>
  </ul>
</template>

<script>
import axios from "axios"
import Config from "../../config.json"
import { library } from "@fortawesome/fontawesome-svg-core"
import { faUsers, faUser } from "@fortawesome/free-solid-svg-icons"

library.add(faUsers, faUser)

// read props groupid
export default {
  data() {
    return {
      current_group: null,
      group_id: null,
    }
  },
  props: {
    groups: {
      type: Array,
      default: [],
    },
  },
  async created() {
    document.title = "List Vtubers - Vtbot"

    // make new promise when this.groups is not empty
    await new Promise((resolve) => {
      this.$watch(
        () => this.groups,
        () => {
          if (this.groups.length > 0) {
            resolve()
          }
        },
        { immediate: true }
      )
    })

    this.$watch(
      () => this.$route.params,
      () => (this.group_id = this.$route.params?.id || null),
      { immediate: true }
    )

    this.$watch(
      () => this.group_id,
      async () => {
        this.current_group = await this.groups.find(
          (group) => group.ID == this.$route.params.id
        )

        if (this.current_group) {
          document.title = `${
            this.current_group?.GroupName.charAt(0).toUpperCase() +
            this.current_group?.GroupName.slice(1).replace("_", " ")
          } - List Vtubers`
        } else {
          document.title = "List Vtubers - Vtbot"
        }

        if (this.$route.path.includes("/vtubers")) {
          if (this.current_group)
            console.log(`Get Group: ${this.current_group.GroupName}`)
          else console.log(`Cannot get group`)
        }
      },
      { immediate: true }
    )
  },
  methods: {},
}
</script>
