<script setup lang="ts">
import { ref, onMounted, computed, watch } from 'vue';
import { useI18n } from 'vue-i18n';
// @ts-ignore
import { QueryHistoryByDSL } from '../../../../bindings/github.com/yhy0/ChYing/app.js';

const { t } = useI18n();
const emit = defineEmits(['search-results', 'clear-search', 'notify']);

const dslQuery = ref('');
const showSuggestions = ref(false);
const suggestions = ref<Array<{value: string, description: string, type: string}>>([]);
const searchHistory = ref<string[]>([]);
const MAX_HISTORY = 10;
// 键盘导航：当前选中的建议索引，-1 表示无选中
const selectedIndex = ref(-1);
const inputRef = ref<HTMLInputElement | null>(null);

// 内置字段提示
const builtinFields = [
  { value: 'method', description: t('modules.proxy.dsl.field_method') },
  { value: 'status', description: t('modules.proxy.dsl.field_status') },
  { value: 'path', description: t('modules.proxy.dsl.field_path') },
  { value: 'host', description: t('modules.proxy.dsl.field_host') },
  { value: 'url', description: t('modules.proxy.dsl.field_url') },
  { value: 'content_type', description: t('modules.proxy.dsl.field_content_type') },
  { value: 'request_body', description: t('modules.proxy.dsl.field_request_body') },
  { value: 'response_body', description: t('modules.proxy.dsl.field_response_body') },
  { value: 'id', description: t('modules.proxy.dsl.field_id') },
  { value: 'length', description: t('modules.proxy.dsl.field_length') },
  { value: 'response', description: t('modules.proxy.dsl.field_response') },
  { value: 'request', description: t('modules.proxy.dsl.field_request') },
];

// 运算符提示
const operators = [
  { value: '== ', description: t('modules.proxy.dsl.function_equals'), type: 'operator' },
  { value: '!= ', description: t('modules.proxy.dsl.function_not_equals'), type: 'operator' },
  { value: '&& ', description: t('modules.proxy.dsl.function_and'), type: 'operator' },
  { value: '|| ', description: t('modules.proxy.dsl.function_or'), type: 'operator' },
];

// 函数提示
const builtinFunctions = [
  { value: 'contains(', description: t('modules.proxy.dsl.function_contains'), type: 'function' },
  { value: 'regex(', description: t('modules.proxy.dsl.function_regex'), type: 'function' },
];

// 示例查询表达式
const dslExamples = [
  { name: t('modules.proxy.dsl.example_contains'), query: 'contains(request, "admin")' },
  { name: t('modules.proxy.dsl.example_status'), query: 'status == "200"' },
  { name: t('modules.proxy.dsl.example_content_type'), query: 'contains(content_type, "application/json")' },
  { name: t('modules.proxy.dsl.example_path'), query: 'contains(path, "/api/")' },
  { name: t('modules.proxy.dsl.example_method'), query: 'method == "POST"' },
  { name: t('modules.proxy.dsl.example_response'), query: 'contains(response_body, "error")' },
  { name: t('modules.proxy.dsl.example_combined'), query: 'method == "GET" && status == "200"' }
];

// 默认建议列表（空输入时显示）
const defaultSuggestions = computed(() => [
  { value: 'status == "200"', description: t('modules.proxy.dsl.default_find_success'), type: 'example' },
  { value: 'contains(path, "/api/")', description: t('modules.proxy.dsl.default_find_api'), type: 'example' },
  { value: 'method == "POST"', description: t('modules.proxy.dsl.default_find_post'), type: 'example' },
]);

// 上下文类型
type InputContext = 'empty' | 'field' | 'after_field' | 'after_operator' | 'after_func_open' | 'after_comma' | 'in_string' | 'after_close';

// 分析光标前的输入上下文
function getInputContext(text: string): { context: InputContext; currentWord: string; wordStart: number } {
  const trimmed = text.trimEnd();

  // 空输入
  if (!trimmed) {
    return { context: 'empty', currentWord: '', wordStart: 0 };
  }

  // 在引号内 — 不提示
  let inDouble = false;
  for (const ch of trimmed) {
    if (ch === '"') inDouble = !inDouble;
  }
  if (inDouble) {
    return { context: 'in_string', currentWord: '', wordStart: text.length };
  }

  // 取光标前最后一个有意义 token
  const tail = trimmed;

  // 刚关闭引号或右括号后 → 提示运算符/逻辑符
  if (/[")]\s*$/.test(tail)) {
    return { context: 'after_close', currentWord: '', wordStart: text.length };
  }

  // 在逗号后（函数参数分隔符） → 提示引号开头的值
  if (/,\s*$/.test(tail)) {
    return { context: 'after_comma', currentWord: '', wordStart: text.length };
  }

  // 在运算符（==, !=）后 → 提示引号开头的值
  if (/[=!]=\s*$/.test(tail)) {
    return { context: 'after_operator', currentWord: '', wordStart: text.length };
  }

  // 在左括号后（函数参数起始） → 提示字段
  if (/\(\s*$/.test(tail)) {
    return { context: 'after_func_open', currentWord: '', wordStart: text.length };
  }

  // 在 && 或 || 后 → 提示字段/函数（新表达式起始）
  if (/(\&\&|\|\|)\s*$/.test(tail)) {
    return { context: 'field', currentWord: '', wordStart: text.length };
  }

  // 正在输入一个单词（字段名或函数名）
  const wordMatch = tail.match(/([a-zA-Z_][a-zA-Z0-9_]*)$/);
  if (wordMatch) {
    const currentWord = wordMatch[1];
    const wordStart = text.length - currentWord.length;
    // 判断这个单词前面是什么
    const before = tail.substring(0, tail.length - currentWord.length).trimEnd();
    if (!before || /(\&\&|\|\||\()\s*$/.test(before)) {
      // 在表达式起始/逻辑符后/括号后输入 → 输入字段或函数名
      return { context: 'field', currentWord, wordStart };
    }
    // 在字段名后继续输入 → 仍在输入字段
    return { context: 'field', currentWord, wordStart };
  }

  // 一个完整字段名后跟空格 → 提示运算符
  if (/[a-zA-Z_][a-zA-Z0-9_]*\s+$/.test(tail)) {
    return { context: 'after_field', currentWord: '', wordStart: text.length };
  }

  return { context: 'field', currentWord: '', wordStart: text.length };
}

// 更新建议列表
const updateSuggestions = () => {
  showSuggestions.value = true;
  selectedIndex.value = -1;

  const text = dslQuery.value;
  const { context, currentWord } = getInputContext(text);

  let result: Array<{value: string, description: string, type: string}> = [];

  switch (context) {
    case 'empty':
      // 空输入：显示历史记录 + 示例 + 常用字段/函数
      result = [
        ...searchHistory.value.slice(0, 3).map(h => ({
          value: h, description: t('modules.proxy.dsl.previous_search'), type: 'history'
        })),
        ...defaultSuggestions.value,
        ...builtinFunctions,
      ];
      break;

    case 'field':
    case 'after_func_open':
      // 输入字段名或函数名
      {
        const lower = currentWord.toLowerCase();
        const fields = builtinFields
          .filter(f => !lower || f.value.includes(lower))
          .map(f => ({ value: f.value, description: f.description, type: 'field' }));
        const funcs = builtinFunctions
          .filter(f => !lower || f.value.toLowerCase().includes(lower))
          .map(f => ({ ...f }));
        result = [...fields, ...funcs];
      }
      break;

    case 'after_field':
      // 字段名后 → 提示运算符
      result = [...operators];
      break;

    case 'after_operator':
      // 运算符后 → 提示带引号的值模板
      result = [
        { value: '"200"', description: t('modules.proxy.dsl.status_200'), type: 'value' },
        { value: '"GET"', description: t('modules.proxy.dsl.method_get'), type: 'value' },
        { value: '"POST"', description: t('modules.proxy.dsl.method_post'), type: 'value' },
        { value: '"application/json"', description: t('modules.proxy.dsl.json_type'), type: 'value' },
      ];
      break;

    case 'after_comma':
      // 函数逗号后 → 提示带引号的值
      result = [
        { value: ' "', description: t('modules.proxy.dsl.enter_match_value'), type: 'value' },
      ];
      break;

    case 'after_close':
      // 表达式闭合后 → 提示逻辑运算符
      result = [
        { value: ' && ', description: t('modules.proxy.dsl.function_and'), type: 'operator' },
        { value: ' || ', description: t('modules.proxy.dsl.function_or'), type: 'operator' },
      ];
      break;

    case 'in_string':
      // 在引号内 → 不提示
      result = [];
      break;
  }

  suggestions.value = result.slice(0, 8);
};

// 应用选中的建议
const applySuggestion = (suggestion: { value: string, type: string }) => {
  if (suggestion.type === 'history' || suggestion.type === 'example') {
    // 历史记录和示例直接替换整个查询
    dslQuery.value = suggestion.value;
  } else {
    const { currentWord, wordStart } = getInputContext(dslQuery.value);
    // 替换当前正在输入的单词
    const prefix = dslQuery.value.substring(0, wordStart);
    const val = suggestion.value;
    // 函数类型不追加空格（光标要在括号内），其他已自带空格的不追加
    const suffix = (suggestion.type === 'function' || val.endsWith(' ') || val.endsWith('"')) ? '' : ' ';
    dslQuery.value = prefix + val + suffix;
  }

  showSuggestions.value = false;
  selectedIndex.value = -1;

  // 聚焦回输入框
  const el = inputRef.value;
  if (el) {
    el.focus();
    // 函数类型：光标放在括号内
    if (suggestion.type === 'function') {
      const pos = dslQuery.value.length;
      setTimeout(() => { el.selectionStart = el.selectionEnd = pos; }, 0);
    }
  }
};

// 是否显示帮助信息
const showHelp = ref(false);
// 是否正在加载
const isLoading = ref(false);

// 执行DSL查询
const executeDSLQuery = async () => {
  if (dslQuery.value.trim() === '') {
    // 如果DSL为空，清除过滤器
    emit('clear-search');
    return;
  }
  
  try {
    isLoading.value = true;
     
    // 检查引号是否平衡
    const queryString = dslQuery.value.trim();
    const doubleQuotes = (queryString.match(/"/g) || []).length;
    const singleQuotes = (queryString.match(/'/g) || []).length;
     
    if (doubleQuotes % 2 !== 0 || singleQuotes % 2 !== 0) {
      throw new Error(t('modules.proxy.dsl.unbalanced_quotes'));
    }
    
    // 将查询添加到历史记录
    addToSearchHistory(queryString);
     
    // 调用后端API执行DSL查询
    const response = await QueryHistoryByDSL(queryString);
    
    // 检查是否有错误
    if (response.error) {
      throw new Error(response.error);
    }
    
    const results = response.data || [];
     
    // 将查询结果发送给父组件
    emit('search-results', results);
     
    // 显示通知
    emit('notify', {
      message: t('modules.proxy.dsl.search_complete', { count: results.length }),
      type: 'success'
    });
  } catch (error: any) { // 使用any类型处理未知错误类型
    console.error('DSL查询错误:', error);
    // 显示错误通知
    emit('notify', {
      message: t('modules.proxy.dsl.search_error', { error: error.toString() }),
      type: 'error'
    });
  } finally {
    isLoading.value = false;
    showSuggestions.value = false;
  }
};

// 添加到搜索历史
const addToSearchHistory = (query: string) => {
  // 如果已经存在，先移除旧记录
  const index = searchHistory.value.indexOf(query);
  if (index > -1) {
    searchHistory.value.splice(index, 1);
  }
  
  // 添加到历史记录开头
  searchHistory.value.unshift(query);
  
  // 限制历史记录数量
  if (searchHistory.value.length > MAX_HISTORY) {
    searchHistory.value = searchHistory.value.slice(0, MAX_HISTORY);
  }
  
  // 保存到 localStorage
  localStorage.setItem('dsl_search_history', JSON.stringify(searchHistory.value));
};

// 清除查询
const clearQuery = () => {
  dslQuery.value = '';
  showSuggestions.value = false;
  emit('clear-search');
};

// 选择示例查询
const selectExample = (query: string) => {
  dslQuery.value = query;
  showSuggestions.value = false;
};

// 输入框焦点事件
const handleInputFocus = () => {
  updateSuggestions();
};

// 输入事件
const handleInput = () => {
  updateSuggestions();
};

// 输入框失去焦点事件
const handleInputBlur = () => {
  // 延迟关闭建议，以便可以点击建议
  setTimeout(() => {
    showSuggestions.value = false;
    selectedIndex.value = -1;
  }, 200);
};

// 输入框键盘事件
const handleInputKeydown = (event: KeyboardEvent) => {
  if (!shouldShowSuggestions.value) {
    if (event.key === 'Enter') executeDSLQuery();
    return;
  }

  switch (event.key) {
    case 'ArrowDown':
      event.preventDefault();
      selectedIndex.value = (selectedIndex.value + 1) % suggestions.value.length;
      break;
    case 'ArrowUp':
      event.preventDefault();
      selectedIndex.value = selectedIndex.value <= 0
        ? suggestions.value.length - 1
        : selectedIndex.value - 1;
      break;
    case 'Tab':
    case 'Enter':
      if (selectedIndex.value >= 0 && selectedIndex.value < suggestions.value.length) {
        event.preventDefault();
        applySuggestion(suggestions.value[selectedIndex.value]);
      } else if (event.key === 'Enter') {
        executeDSLQuery();
      }
      break;
    case 'Escape':
      showSuggestions.value = false;
      selectedIndex.value = -1;
      break;
  }
};

// 初始化时从localStorage加载搜索历史
onMounted(() => {
  // 从localStorage加载搜索历史
  const savedHistory = localStorage.getItem('dsl_search_history');
  if (savedHistory) {
    try {
      searchHistory.value = JSON.parse(savedHistory);
    } catch (e) {
      console.error('Failed to parse search history:', e);
    }
  }
  
  // 初始化建议列表为默认建议
  suggestions.value = defaultSuggestions.value;
});

// 监听输入变化
watch(dslQuery, () => {
  if (showSuggestions.value) {
    updateSuggestions();
  }
});

// 计算显示建议的条件
const shouldShowSuggestions = computed(() => {
  const result = showSuggestions.value && suggestions.value.length > 0;
  return result;
});
</script>

<template>
  <div class="dsl-search-container">
    <div class="dsl-search-input-wrapper">
      <input
        ref="inputRef"
        type="text"
        v-model="dslQuery"
        :placeholder="t('modules.proxy.dsl.query_placeholder')"
        class="dsl-search-input"
        @input="handleInput"
        @focus="handleInputFocus"
        @blur="handleInputBlur"
        @keydown="handleInputKeydown"
        spellcheck="false"
      />

      <!-- 输入建议下拉列表 (使用计算属性) -->
      <div 
        class="dsl-suggestions" 
        v-if="shouldShowSuggestions"
      >
        <div
          v-for="(suggestion, index) in suggestions"
          :key="suggestion.value + index"
          class="dsl-suggestion-item"
          :class="{ 'dsl-suggestion-selected': index === selectedIndex }"
          @mousedown.prevent="applySuggestion(suggestion)"
          @mouseenter="selectedIndex = index"
        >
          <div class="dsl-suggestion-value">
            <span
              class="dsl-suggestion-icon"
              :class="{
                'field-icon': suggestion.type === 'field',
                'function-icon': suggestion.type === 'function',
                'history-icon': suggestion.type === 'history',
                'example-icon': suggestion.type === 'example',
                'operator-icon': suggestion.type === 'operator',
                'value-icon': suggestion.type === 'value'
              }"
            >
              <i
                class="bx"
                :class="{
                  'bx-box': suggestion.type === 'field',
                  'bx-code-alt': suggestion.type === 'function',
                  'bx-history': suggestion.type === 'history',
                  'bx-bulb': suggestion.type === 'example',
                  'bx-math': suggestion.type === 'operator',
                  'bx-text': suggestion.type === 'value'
                }"
              ></i>
            </span>
            <span>{{ suggestion.value }}</span>
          </div>
          <div class="dsl-suggestion-description">{{ suggestion.description }}</div>
        </div>
      </div>
      
      <div class="dsl-search-actions">
        <button 
          class="help-button" 
          @click="showHelp = !showHelp"
          :title="t('modules.proxy.dsl.toggle_help')"
        >
          <i class="bx bx-help-circle"></i>
        </button>
        <button 
          class="dsl-search-button" 
          @click="executeDSLQuery"
          :disabled="isLoading"
          :title="t('modules.proxy.dsl.search')"
        >
          <span v-if="isLoading">
            <i class="bx bx-loader bx-spin"></i>
          </span>
          <span v-else>
            <i class="bx bx-search"></i>
          </span>
        </button>
        <button 
          class="dsl-clear-button" 
          @click="clearQuery"
          :disabled="isLoading || !dslQuery"
          :title="t('modules.proxy.dsl.clear')"
        >
          <i class="bx bx-x"></i>
        </button>
      </div>
    </div>
    
    <!-- DSL帮助信息悬浮窗 -->
    <div v-if="showHelp" class="dsl-help-popup">
      <div class="dsl-help-popup-header">
        <h4>{{ t('modules.proxy.dsl.help_title') }}</h4>
        <button @click="showHelp = false" class="dsl-help-close">
          <i class="bx bx-x"></i>
        </button>
      </div>
      <div class="dsl-help-popup-content">
        <p>{{ t('modules.proxy.dsl.help_description') }}</p>
        
        <h5>{{ t('modules.proxy.dsl.examples_title') }}</h5>
        <div class="dsl-examples">
          <div 
            v-for="example in dslExamples" 
            :key="example.query"
            class="dsl-example"
            @click="selectExample(example.query)"
          >
            <div class="dsl-example-name">{{ example.name }}</div>
            <div class="dsl-example-query">{{ example.query }}</div>
          </div>
        </div>
        
        <h5>{{ t('modules.proxy.dsl.available_fields') }}</h5>
        <ul class="dsl-fields-list">
          <li><code>id</code> - {{ t('modules.proxy.dsl.field_id') }}</li>
          <li><code>url</code> - {{ t('modules.proxy.dsl.field_url') }}</li>
          <li><code>path</code> - {{ t('modules.proxy.dsl.field_path') }}</li>
          <li><code>method</code> - {{ t('modules.proxy.dsl.field_method') }}</li>
          <li><code>host</code> - {{ t('modules.proxy.dsl.field_host') }}</li>
          <li><code>status</code> - {{ t('modules.proxy.dsl.field_status') }}</li>
          <li><code>length</code> - {{ t('modules.proxy.dsl.field_length') }}</li>
          <li><code>content_type</code> - {{ t('modules.proxy.dsl.field_content_type') }}</li>
          <li><code>request</code> - {{ t('modules.proxy.dsl.field_request') }}</li>
          <li><code>request_body</code> - {{ t('modules.proxy.dsl.field_request_body') }}</li>
          <li><code>response</code> - {{ t('modules.proxy.dsl.field_response') }}</li>
          <li><code>response_body</code> - {{ t('modules.proxy.dsl.field_response_body') }}</li>
          <li><code>status_reason</code> - {{ t('modules.proxy.dsl.field_status_reason') }}</li>
        </ul>
        
        <h5>{{ t('modules.proxy.dsl.functions_title') }}</h5>
        <ul class="dsl-functions-list">
          <li><code>contains(field, "value")</code> - {{ t('modules.proxy.dsl.function_contains') }}</li>
          <li><code>regex(field, "pattern")</code> - {{ t('modules.proxy.dsl.function_regex') }}</li>
          <li><code>field == "value"</code> - {{ t('modules.proxy.dsl.function_equals') }}</li>
          <li><code>field != "value"</code> - {{ t('modules.proxy.dsl.function_not_equals') }}</li>
          <li><code>expression1 && expression2</code> - {{ t('modules.proxy.dsl.function_and') }}</li>
          <li><code>expression1 || expression2</code> - {{ t('modules.proxy.dsl.function_or') }}</li>
        </ul>
      </div>
    </div>
  </div>
</template>

<style scoped>
/* 使用外部样式文件定义样式 */
</style> 