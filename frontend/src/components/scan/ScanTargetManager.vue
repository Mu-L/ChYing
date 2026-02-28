<script setup lang="ts">
import { ref, reactive, computed, onMounted, nextTick } from 'vue';
import { useI18n } from 'vue-i18n';
import type {
  ScanTarget,
  ScanTargetStatus,
  ScanTargetType,
  ScanConfig,
  ScanStatistics,
  AddScanTargetParams,
  BatchAddScanTargetsParams,
  SchedulerStatus
} from '../../types/scanTarget';
import {
  getStatusOptions,
  getTypeOptions,
  getPriorityOptions,
  getScheduleTypeOptions,
  STATUS_COLOR_MAP,
  getStatusTextMap,
  getTypeTextMap
} from '../../types/scanTarget';

const { t, locale } = useI18n();

// 响应式选项（computed 包裹以便语言切换时自动更新）
const statusOptions = computed(() => getStatusOptions());
const typeOptions = computed(() => getTypeOptions());
const priorityOptions = computed(() => getPriorityOptions());
const scheduleTypeOptions = computed(() => getScheduleTypeOptions());

// 响应式数据
const targets = ref<ScanTarget[]>([]);
const total = ref(0);
const loading = ref(false);
const statistics = ref<ScanStatistics | null>(null);
const schedulerStatus = ref<SchedulerStatus | null>(null);

// 查询参数
const queryParams = reactive({
  status: '' as ScanTargetStatus | '',
  limit: 20,
  offset: 0,
  search: '',
  type: '' as ScanTargetType | ''
});

// 添加目标表单
const addForm = reactive<AddScanTargetParams>({
  name: '',
  type: 'single',
  target: '',
  description: '',
  priority: 5,
  schedule_type: 'once',
  schedule_time: '',
  created_by: t('scan_target.local_user')
});

// 批量添加表单
const batchForm = reactive<BatchAddScanTargetsParams>({
  targets: [],
  target_type: 'single',
  created_by: t('scan_target.local_user')
});

// 弹框状态
const showAddDialog = ref(false);
const showBatchDialog = ref(false);
const showConfigDialog = ref(false);
const showDetailDialog = ref(false);

// 当前选中的目标
const selectedTarget = ref<ScanTarget | null>(null);

// 扫描配置
const scanConfig = ref<ScanConfig | null>(null);

// 批量目标文本
const batchTargetsText = ref('');

// API调用函数（需要根据实际API接口调整）
declare global {
  interface Window {
    go: {
      main: {
        App: {
          GetScanTargets: (status: string, limit: number, offset: number) => Promise<{data?: {targets: ScanTarget[], total: number}, error?: string}>;
          AddScanTarget: (target: AddScanTargetParams) => Promise<{data?: string, error?: string}>;
          BatchAddScanTargets: (targets: string[], targetType: string, createdBy: string) => Promise<{data?: string, error?: string}>;
          UpdateScanTarget: (target: ScanTarget) => Promise<{data?: string, error?: string}>;
          DeleteScanTarget: (id: number) => Promise<{data?: string, error?: string}>;
          UpdateScanTargetStatus: (id: number, status: string, message: string) => Promise<{data?: string, error?: string}>;
          GetScanStatistics: () => Promise<{data?: ScanStatistics, error?: string}>;
          GetDefaultScanConfig: () => Promise<{data?: ScanConfig, error?: string}>;
          GetSchedulerStatus: () => Promise<{data?: SchedulerStatus, error?: string}>;
          StartScheduler: () => Promise<{data?: string, error?: string}>;
          StopScheduler: () => Promise<{data?: string, error?: string}>;
        };
      };
    };
  }
}

// 计算属性
const filteredTargets = computed(() => {
  return targets.value.filter(target => {
    if (queryParams.search && !target.name.toLowerCase().includes(queryParams.search.toLowerCase()) && 
        !target.target.toLowerCase().includes(queryParams.search.toLowerCase())) {
      return false;
    }
    if (queryParams.status && target.status !== queryParams.status) {
      return false;
    }
    if (queryParams.type && target.type !== queryParams.type) {
      return false;
    }
    return true;
  });
});

const paginatedTargets = computed(() => {
  const start = queryParams.offset;
  const end = start + queryParams.limit;
  return filteredTargets.value.slice(start, end);
});

// 方法
const loadTargets = async () => {
  loading.value = true;
  try {
    const result = await window.go.main.App.GetScanTargets(
      queryParams.status, 
      queryParams.limit, 
      queryParams.offset
    );
    
    if (result.error) {
      console.error('加载扫描目标失败:', result.error);
      return;
    }
    
    if (result.data) {
      targets.value = result.data.targets || [];
      total.value = result.data.total || 0;
    }
  } catch (error) {
    console.error('加载扫描目标失败:', error);
  } finally {
    loading.value = false;
  }
};

const loadStatistics = async () => {
  try {
    const result = await window.go.main.App.GetScanStatistics();
    if (result.data) {
      statistics.value = result.data;
    }
  } catch (error) {
    console.error('加载统计信息失败:', error);
  }
};

const loadSchedulerStatus = async () => {
  try {
    const result = await window.go.main.App.GetSchedulerStatus();
    if (result.data) {
      schedulerStatus.value = result.data;
    }
  } catch (error) {
    console.error('加载调度器状态失败:', error);
  }
};

const loadDefaultConfig = async () => {
  try {
    const result = await window.go.main.App.GetDefaultScanConfig();
    if (result.data) {
      scanConfig.value = result.data;
    }
  } catch (error) {
    console.error('加载默认配置失败:', error);
  }
};

const addTarget = async () => {
  try {
    const result = await window.go.main.App.AddScanTarget(addForm);
    if (result.error) {
      alert(t('scan_target.add_failed', { error: result.error }));
      return;
    }

    showAddDialog.value = false;
    resetAddForm();
    await loadTargets();
    await loadStatistics();
  } catch (error) {
    console.error('添加目标失败:', error);
    alert(t('scan_target.add_failed', { error }));
  }
};

const batchAddTargets = async () => {
  const targetList = batchTargetsText.value.split('\n')
    .map(line => line.trim())
    .filter(line => line.length > 0);

  if (targetList.length === 0) {
    alert(t('scan_target.enter_target_list'));
    return;
  }

  try {
    const result = await window.go.main.App.BatchAddScanTargets(
      targetList,
      batchForm.target_type,
      batchForm.created_by
    );

    if (result.error) {
      alert(t('scan_target.batch_add_failed', { error: result.error }));
      return;
    }

    showBatchDialog.value = false;
    resetBatchForm();
    await loadTargets();
    await loadStatistics();
  } catch (error) {
    console.error('批量添加失败:', error);
    alert(t('scan_target.batch_add_failed', { error }));
  }
};

const deleteTarget = async (target: ScanTarget) => {
  if (!confirm(t('scan_target.confirm_delete', { name: target.name }))) {
    return;
  }

  try {
    const result = await window.go.main.App.DeleteScanTarget(target.id);
    if (result.error) {
      alert(t('scan_target.delete_failed', { error: result.error }));
      return;
    }

    await loadTargets();
    await loadStatistics();
  } catch (error) {
    console.error('删除目标失败:', error);
    alert(t('scan_target.delete_failed', { error }));
  }
};

const updateTargetStatus = async (target: ScanTarget, status: ScanTargetStatus) => {
  try {
    const result = await window.go.main.App.UpdateScanTargetStatus(target.id, status, '');
    if (result.error) {
      alert(t('scan_target.update_status_failed', { error: result.error }));
      return;
    }

    await loadTargets();
  } catch (error) {
    console.error('更新状态失败:', error);
    alert(t('scan_target.update_status_failed', { error }));
  }
};

const startScheduler = async () => {
  try {
    const result = await window.go.main.App.StartScheduler();
    if (result.error) {
      alert(t('scan_target.start_scheduler_failed', { error: result.error }));
      return;
    }

    await loadSchedulerStatus();
  } catch (error) {
    console.error('启动调度器失败:', error);
    alert(t('scan_target.start_scheduler_failed', { error }));
  }
};

const stopScheduler = async () => {
  try {
    const result = await window.go.main.App.StopScheduler();
    if (result.error) {
      alert(t('scan_target.stop_scheduler_failed', { error: result.error }));
      return;
    }

    await loadSchedulerStatus();
  } catch (error) {
    console.error('停止调度器失败:', error);
    alert(t('scan_target.stop_scheduler_failed', { error }));
  }
};

const showTargetDetail = (target: ScanTarget) => {
  selectedTarget.value = target;
  showDetailDialog.value = true;
};

const resetAddForm = () => {
  Object.assign(addForm, {
    name: '',
    type: 'single',
    target: '',
    description: '',
    priority: 5,
    schedule_type: 'once',
    schedule_time: '',
    created_by: t('scan_target.local_user')
  });
};

const resetBatchForm = () => {
  Object.assign(batchForm, {
    targets: [],
    target_type: 'single',
    created_by: t('scan_target.local_user')
  });
  batchTargetsText.value = '';
};

const formatDuration = (seconds: number): string => {
  if (seconds < 60) return t('scan_target.duration.seconds', { n: seconds });
  if (seconds < 3600) return t('scan_target.duration.minutes', { m: Math.floor(seconds / 60), s: seconds % 60 });
  return t('scan_target.duration.hours', { h: Math.floor(seconds / 3600), m: Math.floor((seconds % 3600) / 60) });
};

const formatDateTime = (dateStr: string): string => {
  if (!dateStr) return '-';
  return new Date(dateStr).toLocaleString(locale.value === 'zh' ? 'zh-CN' : 'en-US');
};

const getStatusColor = (status: ScanTargetStatus): string => {
  return STATUS_COLOR_MAP[status] || 'gray';
};

const getStatusText = (status: ScanTargetStatus): string => {
  return getStatusTextMap(status);
};

const getTypeText = (type: ScanTargetType): string => {
  return getTypeTextMap(type);
};

// 生命周期
onMounted(async () => {
  await Promise.all([
    loadTargets(),
    loadStatistics(),
    loadSchedulerStatus(),
    loadDefaultConfig()
  ]);
  
  // 定期刷新调度器状态
  setInterval(loadSchedulerStatus, 5000);
});
</script>

<template>
  <div class="scan-target-manager h-full flex flex-col bg-white dark:bg-gray-900">
    <!-- 头部工具栏 -->
    <div class="header-toolbar border-b border-gray-200 dark:border-gray-700 p-4">
      <div class="flex items-center justify-between">
        <div class="flex items-center space-x-4">
          <h2 class="text-lg font-semibold text-gray-900 dark:text-white">{{ t('scan_target.title') }}</h2>

          <!-- 统计信息 -->
          <div v-if="statistics" class="flex items-center space-x-4 text-sm text-gray-600 dark:text-gray-300">
            <span>{{ t('scan_target.total', { count: statistics.total }) }}</span>
            <span>{{ t('scan_target.today', { count: statistics.today }) }}</span>
          </div>
        </div>
        
        <div class="flex items-center space-x-3">
          <!-- 调度器状态 -->
          <div v-if="schedulerStatus" class="flex items-center space-x-2">
            <div class="flex items-center">
              <div 
                :class="[
                  'w-2 h-2 rounded-full mr-2',
                  schedulerStatus.running ? 'bg-green-500' : 'bg-red-500'
                ]"
              ></div>
              <span class="text-sm text-gray-600 dark:text-gray-300">
                {{ t('scan_target.scheduler') }}: {{ schedulerStatus.running ? t('scan_target.scheduler_running') : t('scan_target.scheduler_stopped') }}
              </span>
            </div>
            <button
              @click="schedulerStatus.running ? stopScheduler() : startScheduler()"
              class="px-3 py-1 text-xs rounded"
              :class="[
                schedulerStatus.running 
                  ? 'bg-red-100 text-red-800 hover:bg-red-200 dark:bg-red-900 dark:text-red-200' 
                  : 'bg-green-100 text-green-800 hover:bg-green-200 dark:bg-green-900 dark:text-green-200'
              ]"
            >
              {{ schedulerStatus.running ? t('scan_target.stop_scheduler') : t('scan_target.start_scheduler') }}
            </button>
          </div>
          
          <!-- 操作按钮 -->
          <button
            @click="showAddDialog = true"
            class="btn btn-primary"
          >
            <i class="bx bx-plus mr-1"></i>
            {{ t('scan_target.add_target') }}
          </button>
          
          <button
            @click="showBatchDialog = true"
            class="btn btn-secondary"
          >
            <i class="bx bx-list-plus mr-1"></i>
            {{ t('scan_target.batch_add') }}
          </button>
          
          <button
            @click="loadTargets()"
            class="btn btn-secondary"
            :disabled="loading"
          >
            <i class="bx bx-refresh" :class="{ 'bx-spin': loading }"></i>
          </button>
        </div>
      </div>
      
      <!-- 筛选条件 -->
      <div class="mt-4 flex items-center space-x-4">
        <div class="flex items-center space-x-2">
          <label class="text-sm text-gray-600 dark:text-gray-300">{{ t('scan_target.filter.status') }}</label>
          <select
            v-model="queryParams.status"
            @change="loadTargets()"
            class="form-select"
          >
            <option value="">{{ t('scan_target.filter.all') }}</option>
            <option v-for="option in statusOptions" :key="option.value" :value="option.value">
              {{ option.label }}
            </option>
          </select>
        </div>

        <div class="flex items-center space-x-2">
          <label class="text-sm text-gray-600 dark:text-gray-300">{{ t('scan_target.filter.type') }}</label>
          <select
            v-model="queryParams.type"
            @change="loadTargets()"
            class="form-select"
          >
            <option value="">{{ t('scan_target.filter.all') }}</option>
            <option v-for="option in typeOptions" :key="option.value" :value="option.value">
              {{ option.label }}
            </option>
          </select>
        </div>

        <div class="flex items-center space-x-2 flex-1">
          <label class="text-sm text-gray-600 dark:text-gray-300">{{ t('scan_target.filter.search') }}</label>
          <input
            v-model="queryParams.search"
            @input="loadTargets()"
            type="text"
            :placeholder="t('scan_target.filter.search_placeholder')"
            class="form-input flex-1"
          >
        </div>
      </div>
    </div>

    <!-- 目标列表 -->
    <div class="flex-1 overflow-auto">
      <div v-if="loading" class="flex items-center justify-center h-32">
        <i class="bx bx-loader-alt bx-spin text-2xl text-gray-400"></i>
      </div>
      
      <div v-else-if="paginatedTargets.length === 0" class="empty-state">
        <div class="text-center py-12">
          <i class="bx bx-target-lock text-4xl text-gray-400 mb-4"></i>
          <p class="text-gray-500 dark:text-gray-400 mb-4">{{ t('scan_target.no_targets') }}</p>
          <button @click="showAddDialog = true" class="btn btn-primary">
            {{ t('scan_target.add_first_target') }}
          </button>
        </div>
      </div>
      
      <div v-else class="divide-y divide-gray-200 dark:divide-gray-700">
        <div
          v-for="target in paginatedTargets"
          :key="target.id"
          class="p-4 hover:bg-gray-50 dark:hover:bg-gray-800 cursor-pointer"
          @click="showTargetDetail(target)"
        >
          <div class="flex items-start justify-between">
            <div class="flex-1 min-w-0">
              <!-- 第一行：名称、状态、类型 -->
              <div class="flex items-center mb-2">
                <h3 class="text-base font-medium text-gray-900 dark:text-white truncate mr-3">
                  {{ target.name }}
                </h3>
                
                <span
                  class="inline-flex items-center px-2 py-1 rounded-full text-xs font-medium mr-2"
                  :class="`bg-${getStatusColor(target.status)}-100 text-${getStatusColor(target.status)}-800 dark:bg-${getStatusColor(target.status)}-900 dark:text-${getStatusColor(target.status)}-200`"
                >
                  {{ getStatusText(target.status) }}
                </span>
                
                <span class="inline-flex items-center px-2 py-1 rounded-full text-xs bg-gray-100 text-gray-800 dark:bg-gray-700 dark:text-gray-200">
                  {{ getTypeText(target.type) }}
                </span>
                
                <span v-if="target.priority >= 8" class="inline-flex items-center px-2 py-1 rounded-full text-xs bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200 ml-2">
                  {{ t('scan_target.high_priority') }}
                </span>
              </div>
              
              <!-- 第二行：目标地址 -->
              <p class="text-sm text-gray-600 dark:text-gray-300 mb-2 truncate">
                <i class="bx bx-link-external mr-1"></i>
                {{ target.target }}
              </p>
              
              <!-- 第三行：描述和节点信息 -->
              <div class="flex items-center text-xs text-gray-500 dark:text-gray-400 space-x-4">
                <span v-if="target.description">{{ target.description }}</span>
                <span v-if="target.node_name">
                  <i class="bx bx-server mr-1"></i>
                  {{ target.node_name }}
                </span>
                <span>
                  <i class="bx bx-time mr-1"></i>
                  {{ formatDateTime(target.created_at) }}
                </span>
              </div>
              
              <!-- 进度条（运行中时显示） -->
              <div v-if="target.status === 'running'" class="mt-2">
                <div class="flex items-center justify-between text-xs text-gray-600 dark:text-gray-300 mb-1">
                  <span>{{ t('scan_target.scan_progress') }}</span>
                  <span>{{ target.progress }}%</span>
                </div>
                <div class="w-full bg-gray-200 rounded-full h-2 dark:bg-gray-700">
                  <div
                    class="bg-blue-600 h-2 rounded-full transition-all duration-300"
                    :style="{ width: target.progress + '%' }"
                  ></div>
                </div>
              </div>
              
              <!-- 统计信息（已完成时显示） -->
              <div v-if="target.status === 'completed'" class="mt-2 grid grid-cols-4 gap-4 text-xs">
                <div class="text-center">
                  <div class="text-gray-500 dark:text-gray-400">{{ t('scan_target.hosts') }}</div>
                  <div class="font-medium text-gray-900 dark:text-white">{{ target.found_hosts }}</div>
                </div>
                <div class="text-center">
                  <div class="text-gray-500 dark:text-gray-400">{{ t('scan_target.ports') }}</div>
                  <div class="font-medium text-gray-900 dark:text-white">{{ target.found_ports }}</div>
                </div>
                <div class="text-center">
                  <div class="text-gray-500 dark:text-gray-400">{{ t('scan_target.vulns') }}</div>
                  <div class="font-medium text-red-600 dark:text-red-400">{{ target.found_vulns }}</div>
                </div>
                <div class="text-center">
                  <div class="text-gray-500 dark:text-gray-400">{{ t('scan_target.dirs') }}</div>
                  <div class="font-medium text-gray-900 dark:text-white">{{ target.found_dirs }}</div>
                </div>
              </div>
            </div>
            
            <!-- 操作按钮 -->
            <div class="flex items-center space-x-2 ml-4" @click.stop>
              <button
                v-if="target.status === 'pending'"
                @click="updateTargetStatus(target, 'running')"
                class="btn-icon btn-icon-sm text-green-600 hover:bg-green-100 dark:hover:bg-green-900"
                :title="t('scan_target.start_scan')"
              >
                <i class="bx bx-play"></i>
              </button>

              <button
                v-if="target.status === 'running'"
                @click="updateTargetStatus(target, 'paused')"
                class="btn-icon btn-icon-sm text-yellow-600 hover:bg-yellow-100 dark:hover:bg-yellow-900"
                :title="t('scan_target.pause_scan')"
              >
                <i class="bx bx-pause"></i>
              </button>

              <button
                v-if="target.status === 'failed'"
                @click="updateTargetStatus(target, 'pending')"
                class="btn-icon btn-icon-sm text-blue-600 hover:bg-blue-100 dark:hover:bg-blue-900"
                :title="t('scan_target.retry_scan')"
              >
                <i class="bx bx-refresh"></i>
              </button>

              <button
                @click="deleteTarget(target)"
                class="btn-icon btn-icon-sm text-red-600 hover:bg-red-100 dark:hover:bg-red-900"
                :title="t('common.actions.delete')"
              >
                <i class="bx bx-trash"></i>
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- 分页 -->
    <div v-if="total > queryParams.limit" class="border-t border-gray-200 dark:border-gray-700 px-4 py-3">
      <div class="flex items-center justify-between">
        <div class="text-sm text-gray-700 dark:text-gray-300">
          {{ t('scan_target.pagination.showing', { from: queryParams.offset + 1, to: Math.min(queryParams.offset + queryParams.limit, total), total }) }}
        </div>
        <div class="flex items-center space-x-2">
          <button
            @click="queryParams.offset = Math.max(0, queryParams.offset - queryParams.limit); loadTargets()"
            :disabled="queryParams.offset === 0"
            class="btn btn-secondary btn-sm"
          >
            {{ t('scan_target.pagination.prev') }}
          </button>
          <button
            @click="queryParams.offset += queryParams.limit; loadTargets()"
            :disabled="queryParams.offset + queryParams.limit >= total"
            class="btn btn-secondary btn-sm"
          >
            {{ t('scan_target.pagination.next') }}
          </button>
        </div>
      </div>
    </div>

    <!-- 添加目标对话框 -->
    <div v-if="showAddDialog" class="dialog-overlay" @click.self="showAddDialog = false">
      <div class="dialog">
        <div class="dialog-header">
          <h3>{{ t('scan_target.dialog.add_title') }}</h3>
          <button @click="showAddDialog = false" class="btn-icon">
            <i class="bx bx-x"></i>
          </button>
        </div>

        <div class="dialog-body">
          <form @submit.prevent="addTarget" class="space-y-4">
            <div class="grid grid-cols-2 gap-4">
              <div>
                <label class="form-label">{{ t('scan_target.dialog.target_name') }}</label>
                <input v-model="addForm.name" type="text" required class="form-input" :placeholder="t('scan_target.dialog.target_name_placeholder')">
              </div>
              <div>
                <label class="form-label">{{ t('scan_target.dialog.target_type') }}</label>
                <select v-model="addForm.type" required class="form-select">
                  <option v-for="option in typeOptions" :key="option.value" :value="option.value">
                    {{ option.label }}
                  </option>
                </select>
              </div>
            </div>

            <div>
              <label class="form-label">{{ t('scan_target.dialog.target_address') }}</label>
              <input v-model="addForm.target" type="text" required class="form-input"
                     :placeholder="t('scan_target.dialog.target_address_placeholder')">
            </div>

            <div>
              <label class="form-label">{{ t('scan_target.dialog.description') }}</label>
              <textarea spellcheck="false" v-model="addForm.description" class="form-input" rows="3"
                        :placeholder="t('scan_target.dialog.description_placeholder')"></textarea>
            </div>

            <div class="grid grid-cols-2 gap-4">
              <div>
                <label class="form-label">{{ t('scan_target.dialog.priority') }}</label>
                <select v-model="addForm.priority" class="form-select">
                  <option v-for="option in priorityOptions" :key="option.value" :value="option.value">
                    {{ option.label }}
                  </option>
                </select>
              </div>
              <div>
                <label class="form-label">{{ t('scan_target.dialog.schedule_type') }}</label>
                <select v-model="addForm.schedule_type" class="form-select">
                  <option v-for="option in scheduleTypeOptions" :key="option.value" :value="option.value">
                    {{ option.label }}
                  </option>
                </select>
              </div>
            </div>

            <div class="dialog-footer">
              <button type="button" @click="showAddDialog = false" class="btn btn-secondary">
                {{ t('common.actions.cancel') }}
              </button>
              <button type="submit" class="btn btn-primary">
                {{ t('common.actions.add') }}
              </button>
            </div>
          </form>
        </div>
      </div>
    </div>

    <!-- 批量添加对话框 -->
    <div v-if="showBatchDialog" class="dialog-overlay" @click.self="showBatchDialog = false">
      <div class="dialog">
        <div class="dialog-header">
          <h3>{{ t('scan_target.dialog.batch_title') }}</h3>
          <button @click="showBatchDialog = false" class="btn-icon">
            <i class="bx bx-x"></i>
          </button>
        </div>

        <div class="dialog-body">
          <form @submit.prevent="batchAddTargets" class="space-y-4">
            <div>
              <label class="form-label">{{ t('scan_target.dialog.target_type') }}</label>
              <select v-model="batchForm.target_type" required class="form-select">
                <option v-for="option in typeOptions" :key="option.value" :value="option.value">
                  {{ option.label }}
                </option>
              </select>
            </div>

            <div>
              <label class="form-label">{{ t('scan_target.dialog.target_list') }}</label>
              <textarea spellcheck="false"
                v-model="batchTargetsText"
                required
                class="form-input"
                rows="10"
                :placeholder="t('scan_target.dialog.target_list_placeholder')"
              ></textarea>
              <p class="text-xs text-gray-500 mt-1">{{ t('scan_target.dialog.target_list_hint') }}</p>
            </div>

            <div class="dialog-footer">
              <button type="button" @click="showBatchDialog = false" class="btn btn-secondary">
                {{ t('common.actions.cancel') }}
              </button>
              <button type="submit" class="btn btn-primary">
                {{ t('scan_target.dialog.batch_add_btn') }}
              </button>
            </div>
          </form>
        </div>
      </div>
    </div>

    <!-- 目标详情对话框 -->
    <div v-if="showDetailDialog && selectedTarget" class="dialog-overlay" @click.self="showDetailDialog = false">
      <div class="dialog dialog-lg">
        <div class="dialog-header">
          <h3>{{ t('scan_target.dialog.detail_title') }}</h3>
          <button @click="showDetailDialog = false" class="btn-icon">
            <i class="bx bx-x"></i>
          </button>
        </div>

        <div class="dialog-body">
          <div class="space-y-6">
            <!-- 基本信息 -->
            <div class="grid grid-cols-2 gap-6">
              <div>
                <h4 class="text-sm font-medium text-gray-900 dark:text-white mb-3">{{ t('scan_target.detail.basic_info') }}</h4>
                <div class="space-y-2 text-sm">
                  <div class="flex justify-between">
                    <span class="text-gray-500 dark:text-gray-400">{{ t('scan_target.detail.name') }}</span>
                    <span class="text-gray-900 dark:text-white">{{ selectedTarget.name }}</span>
                  </div>
                  <div class="flex justify-between">
                    <span class="text-gray-500 dark:text-gray-400">{{ t('scan_target.detail.type') }}</span>
                    <span class="text-gray-900 dark:text-white">{{ getTypeText(selectedTarget.type) }}</span>
                  </div>
                  <div class="flex justify-between">
                    <span class="text-gray-500 dark:text-gray-400">{{ t('scan_target.detail.status') }}</span>
                    <span :class="`text-${getStatusColor(selectedTarget.status)}-600`">
                      {{ getStatusText(selectedTarget.status) }}
                    </span>
                  </div>
                  <div class="flex justify-between">
                    <span class="text-gray-500 dark:text-gray-400">{{ t('scan_target.detail.priority') }}</span>
                    <span class="text-gray-900 dark:text-white">{{ selectedTarget.priority }}</span>
                  </div>
                </div>
              </div>

              <div>
                <h4 class="text-sm font-medium text-gray-900 dark:text-white mb-3">{{ t('scan_target.detail.execution_info') }}</h4>
                <div class="space-y-2 text-sm">
                  <div class="flex justify-between">
                    <span class="text-gray-500 dark:text-gray-400">{{ t('scan_target.detail.start_time') }}</span>
                    <span class="text-gray-900 dark:text-white">{{ formatDateTime(selectedTarget.started_at || '') }}</span>
                  </div>
                  <div class="flex justify-between">
                    <span class="text-gray-500 dark:text-gray-400">{{ t('scan_target.detail.end_time') }}</span>
                    <span class="text-gray-900 dark:text-white">{{ formatDateTime(selectedTarget.completed_at || '') }}</span>
                  </div>
                  <div class="flex justify-between">
                    <span class="text-gray-500 dark:text-gray-400">{{ t('scan_target.detail.duration') }}</span>
                    <span class="text-gray-900 dark:text-white">{{ formatDuration(selectedTarget.duration) }}</span>
                  </div>
                  <div class="flex justify-between">
                    <span class="text-gray-500 dark:text-gray-400">{{ t('scan_target.detail.progress') }}</span>
                    <span class="text-gray-900 dark:text-white">{{ selectedTarget.progress }}%</span>
                  </div>
                </div>
              </div>
            </div>

            <!-- 目标地址 -->
            <div>
              <h4 class="text-sm font-medium text-gray-900 dark:text-white mb-3">{{ t('scan_target.detail.target_address') }}</h4>
              <p class="text-sm text-gray-600 dark:text-gray-300 bg-gray-50 dark:bg-gray-800 p-3 rounded">
                {{ selectedTarget.target }}
              </p>
            </div>

            <!-- 扫描结果统计 -->
            <div v-if="selectedTarget.status === 'completed'">
              <h4 class="text-sm font-medium text-gray-900 dark:text-white mb-3">{{ t('scan_target.detail.scan_results') }}</h4>
              <div class="grid grid-cols-4 gap-4">
                <div class="text-center p-3 bg-gray-50 dark:bg-gray-800 rounded">
                  <div class="text-2xl font-bold text-gray-900 dark:text-white">{{ selectedTarget.found_hosts }}</div>
                  <div class="text-xs text-gray-500 dark:text-gray-400">{{ t('scan_target.found_hosts') }}</div>
                </div>
                <div class="text-center p-3 bg-gray-50 dark:bg-gray-800 rounded">
                  <div class="text-2xl font-bold text-gray-900 dark:text-white">{{ selectedTarget.found_ports }}</div>
                  <div class="text-xs text-gray-500 dark:text-gray-400">{{ t('scan_target.found_ports') }}</div>
                </div>
                <div class="text-center p-3 bg-gray-50 dark:bg-gray-800 rounded">
                  <div class="text-2xl font-bold text-red-600 dark:text-red-400">{{ selectedTarget.found_vulns }}</div>
                  <div class="text-xs text-gray-500 dark:text-gray-400">{{ t('scan_target.found_vulns') }}</div>
                </div>
                <div class="text-center p-3 bg-gray-50 dark:bg-gray-800 rounded">
                  <div class="text-2xl font-bold text-gray-900 dark:text-white">{{ selectedTarget.found_dirs }}</div>
                  <div class="text-xs text-gray-500 dark:text-gray-400">{{ t('scan_target.found_dirs') }}</div>
                </div>
              </div>
            </div>

            <!-- 错误信息 -->
            <div v-if="selectedTarget.error_message">
              <h4 class="text-sm font-medium text-gray-900 dark:text-white mb-3">{{ t('scan_target.detail.error_info') }}</h4>
              <p class="text-sm text-red-600 dark:text-red-400 bg-red-50 dark:bg-red-900/20 p-3 rounded">
                {{ selectedTarget.error_message }}
              </p>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
/* 引入已有的样式类 */
</style> 