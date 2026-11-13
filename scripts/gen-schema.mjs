// 从 简历模板.html 的 data-f 占位项生成编辑器字段定义。
// 字段顺序、多行属性、分组全部以模板为准，模板改动后重新运行本脚本即可。
import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const ROOT = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
// 以 docs/ 下的模板原稿为准；assets/ 只是供 go:embed 使用的同步副本。
const srcTpl = path.join(ROOT, 'docs', '简历模板.html');
const tpl = fs.readFileSync(
  fs.existsSync(srcTpl) ? srcTpl : path.join(ROOT, 'assets', '简历模板.html'),
  'utf8',
);
const out = path.join(ROOT, 'assets', 'schema.json');

const elementRe = /<(\w+)([^>]*\bdata-f="([^"]+)"[^>]*)>/g;
const keys = [];
const meta = new Map();
let m;
while ((m = elementRe.exec(tpl)) !== null) {
  const [, , attrs, key] = m;
  if (meta.has(key)) continue;
  keys.push(key);
  meta.set(key, { multiline: /data-multiline/.test(attrs) });
}

const GROUPS = [
  { id: 'personal', title: '个人信息', en: 'Profile' },
  { id: 'advantage', title: '个人优势', en: 'Self-Assessment' },
  { id: 'work', title: '工作经历', en: 'Work Experience' },
  { id: 'project', title: '项目经验', en: 'Project Experience' },
  { id: 'education', title: '教育背景', en: 'Education' },
  { id: 'certificate', title: '资格证书', en: 'Professional Skills' },
];

function groupOf(k) {
  if (/^(name|jobTarget|birth|gender|nativePlace|phone|email|photo)$/.test(k)) return GROUPS[0];
  if (k === 'brief' || k.startsWith('adv')) return GROUPS[1];
  if (k.startsWith('work')) return GROUPS[2];
  if (k.startsWith('proj')) return GROUPS[3];
  if (k.startsWith('edu')) return GROUPS[4];
  return GROUPS[5];
}

const LABELS = {
  name: '姓名', jobTarget: '求职意向', birth: '出生年月', gender: '性别',
  nativePlace: '籍贯', phone: '联系电话', email: '电子邮箱', photo: '证件照',
  brief: '一句话简述（不加粗）', certLines: '证书 / 技能（每行一条）',
};

function labelOf(k) {
  if (LABELS[k]) return LABELS[k];
  let mm;
  if ((mm = /^adv(\d+)(Lead|Text)$/.exec(k))) return `第 ${mm[1]} 条 · ${mm[2] === 'Lead' ? '要点（加粗）' : '描述'}`;
  if ((mm = /^work(\d+)(Period|Company|Dept|Role|Desc)$/.exec(k))) {
    const names = { Period: '起止时间', Company: '公司名称', Dept: '部门', Role: '职位名称', Desc: '工作描述' };
    return `第 ${mm[1]} 段 · ${names[mm[2]]}`;
  }
  if ((mm = /^proj(\d+)(Name|Desc)$/.exec(k))) return `项目 ${mm[1]} · ${mm[2] === 'Name' ? '名称' : '描述'}`;
  if ((mm = /^edu(\d+)(Period|School|Major|Honor|Achievement)$/.exec(k))) {
    const names = { Period: '起止时间', School: '学校名称', Major: '专业 / 学位', Honor: '主要荣誉', Achievement: '学术成果' };
    return `教育 ${mm[1]} · ${names[mm[2]]}`;
  }
  return k;
}

function typeOf(k) {
  if (meta.get(k)?.multiline) return 'lines';
  if (k === 'photo') return 'text';
  if (/(Desc|Text|Honor|Achievement|brief|certLines)$/.test(k)) return 'textarea';
  return 'text';
}

const schema = keys.map((key, i) => {
  const g = groupOf(key);
  return {
    index: i + 1,
    key,
    group: g.id,
    groupTitle: g.title,
    groupEn: g.en,
    label: labelOf(key),
    type: typeOf(key),
    multiline: !!meta.get(key)?.multiline,
  };
});

fs.writeFileSync(out, JSON.stringify(schema, null, 2) + '\n', 'utf8');
const counts = Object.fromEntries(GROUPS.map(g => [g.title, schema.filter(s => s.group === g.id).length]));
console.log(`生成 ${out}`);
console.log(`共 ${schema.length} 个字段：`, counts);
