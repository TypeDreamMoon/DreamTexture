import { createApp } from 'vue'
import { createRouter, createWebHashHistory } from 'vue-router'
import App from './App.vue'
import './style.css'

const router = createRouter({
  history: createWebHashHistory(),
  routes: [
    { path: '/', redirect: '/generate' },
    { path: '/generate', name: 'generate', component: () => import('./views/GenerateView.vue') },
    { path: '/library', name: 'library', component: () => import('./views/LibraryView.vue') },
    { path: '/models', name: 'models', component: () => import('./views/ModelsView.vue') },
    { path: '/workflows', name: 'workflows', component: () => import('./views/WorkflowsView.vue') },
    { path: '/nodes', name: 'nodes', component: () => import('./views/NodesView.vue') },
    { path: '/console', name: 'console', component: () => import('./views/ConsoleView.vue') },
    { path: '/settings', name: 'settings', component: () => import('./views/SettingsView.vue') },
    {
      path: '/material/:id',
      name: 'material',
      component: () => import('./views/MaterialView.vue'),
      props: true,
    },
  ],
})

createApp(App).use(router).mount('#app')
