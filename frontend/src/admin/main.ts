import { createApp } from "vue";
import App from "./App.vue";
import "element-plus/dist/index.css";
import "../styles/global.css";

import ElementPlus from "element-plus";

createApp(App).use(ElementPlus).mount("#app");

