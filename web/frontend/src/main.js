import Vue from 'vue'
import VueI18n from 'vue-i18n'
import App from './App.vue'
import router from './router'
import vuetify from '@/plugins/vuetify'
import utils from './utils'

Vue.config.productionTip = false
Vue.prototype.utils = utils

Vue.use(VueI18n)

const i18n = new VueI18n({
    locale: 'en-US',
    messages: {
        'en-US': require('@/assets/languages/en-US.json')
    }
});

new Vue({
    render: h => h(App),
    router: router,
    i18n,
    vuetify,
}).$mount('#app')
