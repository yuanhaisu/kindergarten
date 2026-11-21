// 兼容既有 import：scene 解析已统一到 scene.mjs（#176 加了 kg_<public_code>）。
// 这里只做转发，调用点无需改动。
export {
  decodeScene,
  parseScenePublicCode,
  parseSceneTaskId
} from "./scene.mjs"
