import { generateAssets } from './lib/assets.mjs';

const data = generateAssets();
console.log('字段定义已生成：' + data.schema.length + ' 个字段');
