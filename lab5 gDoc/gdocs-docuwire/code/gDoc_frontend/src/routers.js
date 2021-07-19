import Hello from './views/Hello.vue'
import LuckySheet from './views/LuckySheet.vue'
import FileList from './views/FileList.vue'
import Log from './views/Log.vue'

const routers = [
  {
    path: '/',
    name: 'Hello',
    component: Hello
  },
  {
    path: '/file/:filename',
    name: 'LuckySheet',
    component: LuckySheet
  },
  {
    path: '/filelist',
    name: 'FileList',
    component: FileList
  },
  {
    path: '/log/:filename',
    name: 'Log',
    component: Log
  }
]
export default routers