import React, {useEffect, useState} from 'react';
import {useTranslation} from 'react-i18next';
import {
  Button,
  Card,
  Checkbox,
  Form,
  Header,
  Icon,
  Input,
  Label,
  Message,
  Modal,
} from 'semantic-ui-react';
import {useNavigate, useParams} from 'react-router-dom';
import {
  API,
  copy,
  getChannelModels,
  showError,
  showInfo,
  showSuccess,
  showWarning,
  verifyJSON,
} from '../../helpers';
import {CHANNEL_OPTIONS} from '../../constants';
import {renderChannelTip} from '../../helpers/render';

const MODEL_MAPPING_EXAMPLE = {
  'gpt-3.5-turbo-0301': 'gpt-3.5-turbo',
  'gpt-4-0314': 'gpt-4',
  'gpt-4-32k-0314': 'gpt-4-32k',
};

const createEmptyKeyRow = () => ({
  id: `new-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`,
  key_value: '',
  status: 'enabled',
  last_error: '',
  last_used_at: '',
  cooldown_until: '',
});

const buildKeyRowFromData = (key = {}) => ({
  id: key.id ?? `key-${key.key_value}`,
  key_value: key.key_value ?? '',
  status: key.status ?? 'enabled',
  last_error: key.last_error ?? '',
  last_used_at: key.last_used_at ?? '',
  cooldown_until: key.cooldown_until ?? '',
});

function type2secretPrompt(type, t) {
  switch (type) {
    case 15:
      return t('channel.edit.key_prompts.zhipu');
    case 18:
      return t('channel.edit.key_prompts.spark');
    case 22:
      return t('channel.edit.key_prompts.fastgpt');
    case 23:
      return t('channel.edit.key_prompts.tencent');
    default:
      return t('channel.edit.key_prompts.default');
  }
}

function buildKeyRows(data) {
  if (Array.isArray(data.keys) && data.keys.length > 0) {
    return data.keys.map((key) => buildKeyRowFromData(key));
  }
  if (data.key) {
    return data.key
      .split('\n')
      .map((keyValue) => keyValue.trim())
      .filter(Boolean)
      .map((keyValue) => ({
        ...createEmptyKeyRow(),
        key_value: keyValue,
      }));
  }
  return [createEmptyKeyRow()];
}

function formatDateTime(value) {
  if (!value) {
    return '';
  }
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return value;
  }
  return date.toLocaleString();
}

const normalizeModelIdPrefix = (value = '') => value.trim().replace(/^-+|-+$/g, '');

const stripModelIdPrefix = (modelName, prefix) => {
  const normalizedPrefix = normalizeModelIdPrefix(prefix);
  if (!normalizedPrefix) {
    return modelName;
  }
  const prefixWithSeparator = `${normalizedPrefix}-`;
  if (modelName.startsWith(prefixWithSeparator)) {
    return modelName.slice(prefixWithSeparator.length);
  }
  return modelName;
};

const applyModelIdPrefix = (modelName, prefix) => {
  const normalizedModelName = modelName.trim();
  const normalizedPrefix = normalizeModelIdPrefix(prefix);
  if (!normalizedModelName || !normalizedPrefix) {
    return normalizedModelName;
  }
  const prefixWithSeparator = `${normalizedPrefix}-`;
  if (normalizedModelName.startsWith(prefixWithSeparator)) {
    return normalizedModelName;
  }
  return `${prefixWithSeparator}${normalizedModelName}`;
};

const mapModelsWithPrefix = (models, prefix, previousPrefix = '') => {
  const seen = new Set();
  return (models || [])
    .map((modelName) =>
      applyModelIdPrefix(stripModelIdPrefix(modelName, previousPrefix), prefix)
    )
    .filter((modelName) => {
      if (!modelName || seen.has(modelName)) {
        return false;
      }
      seen.add(modelName);
      return true;
    });
};

const EditChannel = () => {
  const { t } = useTranslation();
  const params = useParams();
  const navigate = useNavigate();
  const channelId = params.id;
  const isEdit = channelId !== undefined;
  const [loading, setLoading] = useState(isEdit);
  const [recoveringKeyId, setRecoveringKeyId] = useState(null);
  const [modelPickerOpen, setModelPickerOpen] = useState(false);
  const [discoveryLoading, setDiscoveryLoading] = useState(false);
  const [discoveryResult, setDiscoveryResult] = useState(null);
  const [discoverySearch, setDiscoverySearch] = useState('');
  const [selectedDiscoveredModels, setSelectedDiscoveredModels] = useState([]);
  const handleCancel = () => {
    navigate('/channel');
  };

  const originInputs = {
    name: '',
    type: 50,
    key: '',
    base_url: '',
    other: '',
    model_id_prefix: '',
    model_mapping: '',
    system_prompt: '',
    models: [],
    groups: ['default'],
  };
  const [inputs, setInputs] = useState(originInputs);
  const [keyRows, setKeyRows] = useState([createEmptyKeyRow()]);
  const [originModelOptions, setOriginModelOptions] = useState([]);
  const [modelOptions, setModelOptions] = useState([]);
  const [groupOptions, setGroupOptions] = useState([]);
  const [customModel, setCustomModel] = useState('');
  const [config, setConfig] = useState({
    region: '',
    sk: '',
    ak: '',
    user_id: '',
    upstream_proxy: '',
    vertex_ai_project_id: '',
    vertex_ai_adc: '',
    edge_proxy: false,
  });
  const handleInputChange = (e, { name, value }) => {
    if (name === 'model_id_prefix') {
      setInputs((currentInputs) => {
        const nextPrefix = normalizeModelIdPrefix(value);
        return {
          ...currentInputs,
          model_id_prefix: nextPrefix,
          models: mapModelsWithPrefix(
            currentInputs.models,
            nextPrefix,
            currentInputs.model_id_prefix
          ),
        };
      });
      return;
    }
    if (name === 'type') {
      setInputs((currentInputs) => {
        const nextInputs = { ...currentInputs, [name]: value };
        if (currentInputs.models.length === 0) {
          nextInputs.models = mapModelsWithPrefix(
            getChannelModels(value),
            currentInputs.model_id_prefix
          );
        }
        return nextInputs;
      });
      setDiscoveryResult(null);
      return;
    }
    setInputs((inputs) => ({ ...inputs, [name]: value }));
  };

  const handleConfigChange = (e, { name, value, checked }) => {
    setConfig((inputs) => ({ ...inputs, [name]: value !== undefined ? value : checked }));
  };

  const updateKeyRow = (index, patch) => {
    setKeyRows((rows) =>
      rows.map((row, rowIndex) =>
        rowIndex === index ? { ...row, ...patch } : row
      )
    );
  };

  const replaceKeyRowById = (keyId, nextRow) => {
    setKeyRows((rows) =>
      rows.map((row) => (String(row.id) === String(keyId) ? nextRow : row))
    );
  };

  const addKeyRow = () => {
    setKeyRows((rows) => [...rows, createEmptyKeyRow()]);
  };

  const removeKeyRow = (index) => {
    setKeyRows((rows) => {
      const nextRows = rows.filter((_, rowIndex) => rowIndex !== index);
      return nextRows.length > 0 ? nextRows : [createEmptyKeyRow()];
    });
  };

  const moveKeyRow = (index, direction) => {
    setKeyRows((rows) => {
      const nextIndex = index + direction;
      if (nextIndex < 0 || nextIndex >= rows.length) {
        return rows;
      }
      const nextRows = [...rows];
      [nextRows[index], nextRows[nextIndex]] = [
        nextRows[nextIndex],
        nextRows[index],
      ];
      return nextRows;
    });
  };

  const getSubmissionKeyValues = () => {
    if (
      inputs.type === 33 &&
      config.ak.trim() &&
      config.sk.trim() &&
      config.region.trim()
    ) {
      return [`${config.ak}|${config.sk}|${config.region}`];
    }
    if (
      inputs.type === 42 &&
      config.region.trim() &&
      config.vertex_ai_project_id.trim() &&
      config.vertex_ai_adc.trim()
    ) {
      return [
        `${config.region}|${config.vertex_ai_project_id}|${config.vertex_ai_adc}`,
      ];
    }
    const values = [];
    const seen = new Set();
    keyRows.forEach((row) => {
      const value = row.key_value.trim();
      if (!value || seen.has(value)) {
        return;
      }
      seen.add(value);
      values.push(value);
    });
    return values;
  };

  const isVideoTaskChannelV1 = () => inputs.type === 52;

  const requiresDiscoveryBaseURL = () => inputs.type === 8 || inputs.type === 50;

  const canDiscoverModels = () => {
    if (isVideoTaskChannelV1()) {
      return false;
    }
    if (!requiresDiscoveryBaseURL()) {
      return true;
    }
    return Boolean(inputs.base_url && inputs.base_url.trim());
  };

  const getDiscoverySourceText = (source) => {
    switch (source) {
      case 'dynamic':
        return t('channel.edit.discovery.sources.dynamic');
      case 'fallback_static':
        return t('channel.edit.discovery.sources.fallback_static');
      default:
        return t('channel.edit.discovery.sources.manual_only');
    }
  };

  const getDiscoveryWarningMessage = (data) => {
    if (!data) {
      return '';
    }
    const parts = [data.message].filter(Boolean);
    if (data.debug_endpoint) {
      parts.push(`${t('channel.edit.discovery.endpoint')}: ${data.debug_endpoint}`);
    }
    if (data.debug_error) {
      parts.push(`${t('channel.edit.discovery.debug_error')}: ${data.debug_error}`);
    }
    return parts.join(' | ');
  };

  const getKeyStatusColor = (status) => {
    switch (status) {
      case 'cooldown':
        return 'orange';
      case 'disabled':
        return 'red';
      default:
        return 'green';
    }
  };

  const getKeyStatusText = (status) => {
    switch (status) {
      case 'cooldown':
        return t('channel.edit.key_status.cooldown');
      case 'disabled':
        return t('channel.edit.key_status.disabled');
      default:
        return t('channel.edit.key_status.enabled');
    }
  };

  const loadChannel = async () => {
    let res = await API.get(`/api/channel/${channelId}`);
    const { success, message, data } = res.data;
    if (success) {
      if (data.models === '') {
        data.models = [];
      } else {
        data.models = data.models.split(',');
      }
      if (data.group === '') {
        data.groups = [];
      } else {
        data.groups = data.group.split(',');
      }
      if (data.model_mapping !== '') {
        data.model_mapping = JSON.stringify(
          JSON.parse(data.model_mapping),
          null,
          2
        );
      }
      data.model_id_prefix = normalizeModelIdPrefix(data.model_id_prefix || '');
      setInputs(data);
      setKeyRows(buildKeyRows(data));
      if (data.config !== '') {
        setConfig(JSON.parse(data.config));
      }
    } else {
      showError(message);
    }
    setLoading(false);
  };

  const handleEnableKeyV2 = async (row) => {
    if (!isEdit) {
      return;
    }
    const parsedKeyId = Number.parseInt(String(row.id), 10);
    if (!Number.isInteger(parsedKeyId)) {
      showError(t('channel.edit.key_actions.reenable_invalid'));
      return;
    }

    setRecoveringKeyId(parsedKeyId);
    try {
      const res = await API.post(`/api/channel/${channelId}/key/${parsedKeyId}/enable_v2`);
      const { success, message, data } = res.data;
      if (!success) {
        showError(message);
        return;
      }
      replaceKeyRowById(parsedKeyId, buildKeyRowFromData(data));
      showSuccess(t('channel.edit.key_actions.reenable_success'));
    } catch (error) {
      showError(error.message);
    } finally {
      setRecoveringKeyId(null);
    }
  };

  const fetchModels = async () => {
    try {
      let res = await API.get(`/api/channel/models`);
      let localModelOptions = res.data.data.map((model) => ({
        key: model.id,
        text: model.id,
        value: model.id,
      }));
      setOriginModelOptions(localModelOptions);
    } catch (error) {
      showError(error.message);
    }
  };

  const fetchGroups = async () => {
    try {
      let res = await API.get(`/api/group/`);
      setGroupOptions(
        res.data.data.map((group) => ({
          key: group,
          text: group,
          value: group,
        }))
      );
    } catch (error) {
      showError(error.message);
    }
  };

  useEffect(() => {
    const localModelOptionMap = new Map();
    originModelOptions.forEach((option) => {
      const modelName = applyModelIdPrefix(option.value, inputs.model_id_prefix);
      localModelOptionMap.set(modelName, {
        key: modelName,
        text: modelName,
        value: modelName,
      });
    });
    inputs.models.forEach((model) => {
      localModelOptionMap.set(model, {
        key: model,
        text: model,
        value: model,
      });
    });
    setModelOptions(Array.from(localModelOptionMap.values()));
  }, [originModelOptions, inputs.model_id_prefix, inputs.models]);

  useEffect(() => {
    if (isEdit) {
      loadChannel().then();
    } else {
      setInputs((currentInputs) => ({
        ...currentInputs,
        models: mapModelsWithPrefix(
          getChannelModels(currentInputs.type),
          currentInputs.model_id_prefix
        ),
      }));
    }
    fetchModels().then();
    fetchGroups().then();
  }, []);

  const submit = async () => {
    const keyValues = getSubmissionKeyValues();
    const hasSubmissionCredential =
      keyValues.length > 0 || Boolean(inputs.base_url && inputs.base_url.trim());
    if (!isEdit && (inputs.name === '' || !hasSubmissionCredential)) {
      const accessRequiredMessage = t('channel.edit.messages.access_required');
      showInfo(
        accessRequiredMessage === 'channel.edit.messages.access_required'
          ? '请填写渠道名称，并至少提供一种接入信息（API 密钥或 Base URL）！'
          : accessRequiredMessage
      );
      return;
    }
    if (inputs.type !== 43 && inputs.models.length === 0) {
      showInfo(t('channel.edit.messages.models_required'));
      return;
    }
    if (inputs.model_mapping !== '' && !verifyJSON(inputs.model_mapping)) {
      showInfo(t('channel.edit.messages.model_mapping_invalid'));
      return;
    }
    let localInputs = {
      ...inputs,
      key: keyValues[0] || '',
      keys: keyValues,
      model_id_prefix: normalizeModelIdPrefix(inputs.model_id_prefix),
    };
    if (localInputs.base_url && localInputs.base_url.endsWith('/')) {
      localInputs.base_url = localInputs.base_url.slice(
        0,
        localInputs.base_url.length - 1
      );
    }
    if (localInputs.type === 3 && localInputs.other === '') {
      localInputs.other = '2024-03-01-preview';
    }
    let res;
    localInputs.models = localInputs.models.join(',');
    localInputs.group = localInputs.groups.join(',');
    localInputs.config = JSON.stringify(config);
    if (isEdit) {
      res = await API.put(`/api/channel/`, {
        ...localInputs,
        id: parseInt(channelId),
      });
    } else {
      res = await API.post(`/api/channel/`, localInputs);
    }
    const { success, message } = res.data;
    if (success) {
      if (isEdit) {
        showSuccess(t('channel.edit.messages.update_success'));
        loadChannel().then();
      } else {
        showSuccess(t('channel.edit.messages.create_success'));
        setInputs({
          ...originInputs,
          models: mapModelsWithPrefix(
            getChannelModels(originInputs.type),
            originInputs.model_id_prefix
          ),
        });
        setKeyRows([createEmptyKeyRow()]);
        setDiscoveryResult(null);
      }
    } else {
      showError(message);
    }
  };

  const fetchDiscoveredModels = async () => {
    if (isVideoTaskChannelV1()) {
      showInfo(
        'Video Task (V1) requires manual model entry. Dynamic model discovery is not available in V1.'
      );
      return null;
    }
    if (!canDiscoverModels()) {
      showInfo(t('channel.edit.messages.base_url_required'));
      return null;
    }
    const keyValues = getSubmissionKeyValues();
    setDiscoveryLoading(true);
    try {
      const res = await API.post('/api/channel/models/discover', {
        type: inputs.type,
        base_url: inputs.base_url,
        key: keyValues[0],
        keys: keyValues,
        config,
      });
      const { success, message, data } = res.data;
      if (!success) {
        showError(message);
        return null;
      }
      setDiscoveryResult(data);
      if (data.source && data.source !== 'dynamic') {
        showWarning(getDiscoveryWarningMessage(data));
      }
      setOriginModelOptions((options) => {
        const optionMap = new Map(options.map((option) => [option.value, option]));
        data.models.forEach((modelName) => {
          optionMap.set(modelName, {
            key: modelName,
            text: modelName,
            value: modelName,
          });
        });
        return Array.from(optionMap.values());
      });
      return data;
    } catch (error) {
      showError(error.message);
      return null;
    } finally {
      setDiscoveryLoading(false);
    }
  };

  const importAllModels = async () => {
    const data = await fetchDiscoveredModels();
    if (!data) {
      return;
    }
    handleInputChange(null, {
      name: 'models',
      value: mapModelsWithPrefix(data.models, inputs.model_id_prefix),
    });
  };

  const clearDelistedModels = async () => {
    const data = await fetchDiscoveredModels();
    if (!data) {
      return;
    }
    const discoveredModels = new Set(
      mapModelsWithPrefix(data.models, inputs.model_id_prefix)
    );
    const nextModels = inputs.models.filter((modelName) =>
      discoveredModels.has(modelName)
    );
    handleInputChange(null, {
      name: 'models',
      value: nextModels,
    });
    showSuccess(t('channel.edit.messages.clear_delisted_success'));
  };

  const openModelPicker = async () => {
    const data = await fetchDiscoveredModels();
    if (!data) {
      return;
    }
    const discoveredModelDisplayList = mapModelsWithPrefix(
      data.models,
      inputs.model_id_prefix
    );
    setSelectedDiscoveredModels(
      inputs.models.filter((modelName) =>
        discoveredModelDisplayList.includes(modelName)
      )
    );
    setDiscoverySearch('');
    setModelPickerOpen(true);
  };

  const toggleDiscoveredModel = (modelName) => {
    setSelectedDiscoveredModels((models) => {
      if (models.includes(modelName)) {
        return models.filter((item) => item !== modelName);
      }
      return [...models, modelName];
    });
  };

  const confirmSelectedModels = () => {
    handleInputChange(null, {
      name: 'models',
      value: Array.from(new Set([...inputs.models, ...selectedDiscoveredModels])),
    });
    setModelPickerOpen(false);
  };

  const addCustomModel = () => {
    if (customModel.trim() === '') return;
    const modelName = applyModelIdPrefix(customModel, inputs.model_id_prefix);
    if (inputs.models.includes(modelName)) return;
    let localModels = [...inputs.models];
    localModels.push(modelName);
    setCustomModel('');
    handleInputChange(null, { name: 'models', value: localModels });
  };

  const discoveredModelDisplayList = mapModelsWithPrefix(
    discoveryResult?.models || [],
    inputs.model_id_prefix
  );

  return (
      <div className='dashboard-container'>
        <Card fluid className='chart-card'>
          <Card.Content>
          <Card.Header className='header'>
            {isEdit
              ? t('channel.edit.title_edit')
              : t('channel.edit.title_create')}
          </Card.Header>
          <Form loading={loading} autoComplete='new-password'>
            <Form.Field>
              <Form.Select
                label={t('channel.edit.type')}
                name='type'
                required
                search
                options={CHANNEL_OPTIONS}
                value={inputs.type}
                onChange={handleInputChange}
              />
            </Form.Field>
            <Form.Field>
              <Form.Input
                label={t('channel.edit.name')}
                name='name'
                placeholder={t('channel.edit.name_placeholder')}
                onChange={handleInputChange}
                value={inputs.name}
                required
              />
            </Form.Field>
            <Form.Field>
              <Form.Dropdown
                label={t('channel.edit.group')}
                placeholder={t('channel.edit.group_placeholder')}
                name='groups'
                required
                fluid
                multiple
                selection
                allowAdditions
                additionLabel={t('channel.edit.group_addition')}
                onChange={handleInputChange}
                value={inputs.groups}
                autoComplete='new-password'
                options={groupOptions}
              />
            </Form.Field>
            {renderChannelTip(inputs.type)}

            {/* Azure OpenAI specific fields */}
            {inputs.type === 3 && (
              <>
                <Message>
                  注意，<strong>模型部署名称必须和模型名称保持一致</strong>
                  ，因为 Alfred API 会把请求体中的 model
                  参数替换为你的部署名称（模型名称中的点会被剔除），
                  <a
                    target='_blank'
                    href='https://github.com/songquanpeng/one-api/issues/133?notification_referrer_id=NT_kwDOAmJSYrM2NjIwMzI3NDgyOjM5OTk4MDUw#issuecomment-1571602271'
                  >
                    图片演示
                  </a>
                  。
                </Message>
                <Form.Field>
                  <Form.Input
                    label='AZURE_OPENAI_ENDPOINT'
                    name='base_url'
                    placeholder='请输入 AZURE_OPENAI_ENDPOINT，例如：https://docs-test-001.openai.azure.com'
                    onChange={handleInputChange}
                    value={inputs.base_url}
                    autoComplete='new-password'
                  />
                </Form.Field>
                <Form.Field>
                  <Form.Input
                    label='默认 API 版本'
                    name='other'
                    placeholder='请输入默认 API 版本，例如：2024-03-01-preview，该配置可以被实际的请求查询参数所覆盖'
                    onChange={handleInputChange}
                    value={inputs.other}
                    autoComplete='new-password'
                  />
                </Form.Field>
              </>
            )}

            {/* Custom base URL field */}
            {inputs.type === 8 && (
              <Form.Field>
                <Form.Input
                    required
                    label={t('channel.edit.proxy_url')}
                    name='base_url'
                    placeholder={t('channel.edit.proxy_url_placeholder')}
                    onChange={handleInputChange}
                    value={inputs.base_url}
                    autoComplete='new-password'
                />
              </Form.Field>
            )}
            {inputs.type === 50 && (
                <Form.Field>
                  <Form.Input
                      required
                  label={t('channel.edit.base_url')}
                  name='base_url'
                  placeholder={t('channel.edit.base_url_placeholder')}
                  onChange={handleInputChange}
                  value={inputs.base_url}
                  autoComplete='new-password'
                />
              </Form.Field>
            )}
            {inputs.type === 52 && (
              <Form.Field>
                <Form.Input
                  required
                  label='Video Task Base URL'
                  name='base_url'
                  placeholder='Enter the upstream base URL for the task-based video provider'
                  onChange={handleInputChange}
                  value={inputs.base_url}
                  autoComplete='new-password'
                />
              </Form.Field>
            )}

            {inputs.type === 18 && (
              <Form.Field>
                <Form.Input
                  label={t('channel.edit.spark_version')}
                  name='other'
                  placeholder={t('channel.edit.spark_version_placeholder')}
                  onChange={handleInputChange}
                  value={inputs.other}
                  autoComplete='new-password'
                />
              </Form.Field>
            )}
            {inputs.type === 21 && (
              <Form.Field>
                <Form.Input
                  label={t('channel.edit.knowledge_id')}
                  name='other'
                  placeholder={t('channel.edit.knowledge_id_placeholder')}
                  onChange={handleInputChange}
                  value={inputs.other}
                  autoComplete='new-password'
                />
              </Form.Field>
            )}
            {inputs.type === 17 && (
              <Form.Field>
                <Form.Input
                  label={t('channel.edit.plugin_param')}
                  name='other'
                  placeholder={t('channel.edit.plugin_param_placeholder')}
                  onChange={handleInputChange}
                  value={inputs.other}
                  autoComplete='new-password'
                />
              </Form.Field>
            )}
            {inputs.type === 34 && (
              <Message>{t('channel.edit.coze_notice')}</Message>
            )}
            {inputs.type === 40 && (
              <Message>
                {t('channel.edit.douban_notice')}
                <a
                  target='_blank'
                  href='https://console.volcengine.com/ark/region:ark+cn-beijing/endpoint'
                >
                  {t('channel.edit.douban_notice_link')}
                </a>
                {t('channel.edit.douban_notice_2')}
              </Message>
            )}

            <Form.Field>
              <Checkbox
                label='启用 Cloudflare Worker 边缘代理 (Edge Proxy)'
                name='edge_proxy'
                checked={config.edge_proxy === true}
                onChange={handleConfigChange}
              />
            </Form.Field>
            <Form.Field>
              <Form.Input
                label={t('channel.edit.upstream_proxy')}
                name='upstream_proxy'
                placeholder={t('channel.edit.upstream_proxy_placeholder')}
                onChange={handleConfigChange}
                value={config.upstream_proxy || ''}
                autoComplete='new-password'
              />
            </Form.Field>

            {inputs.type !== 43 && (
              <Form.Field>
                <Form.Input
                  label={t('channel.edit.model_id_prefix')}
                  name='model_id_prefix'
                  placeholder={t('channel.edit.model_id_prefix_placeholder')}
                  onChange={handleInputChange}
                  value={inputs.model_id_prefix || ''}
                  autoComplete='new-password'
                />
              </Form.Field>
            )}
            {inputs.type !== 43 && (
              <Form.Field>
                <Form.Dropdown
                  label={t('channel.edit.models')}
                  placeholder={t('channel.edit.models_placeholder')}
                  name='models'
                  required
                  fluid
                  multiple
                  search
                  onLabelClick={(e, { value }) => {
                    copy(value).then();
                  }}
                  selection
                  onChange={handleInputChange}
                  value={inputs.models}
                  autoComplete='new-password'
                  options={modelOptions}
                />
              </Form.Field>
            )}
            {inputs.type !== 43 && (
              <>
                {!isVideoTaskChannelV1() && discoveryResult && (
                  <Message info={discoveryResult.source === 'dynamic'} warning={discoveryResult.source !== 'dynamic'}>
                    <strong>{getDiscoverySourceText(discoveryResult.source)}</strong>
                    {' · '}
                    {discoveryResult.message}
                    {discoveryResult.debug_endpoint && (
                      <div style={{ marginTop: '8px', wordBreak: 'break-all' }}>
                        <strong>{t('channel.edit.discovery.endpoint')}:</strong>
                        {' '}
                        {discoveryResult.debug_endpoint}
                      </div>
                    )}
                    {discoveryResult.debug_error && (
                      <div style={{ marginTop: '8px', wordBreak: 'break-word' }}>
                        <strong>{t('channel.edit.discovery.debug_error')}:</strong>
                        {' '}
                        {discoveryResult.debug_error}
                      </div>
                    )}
                  </Message>
                )}
                {isVideoTaskChannelV1() && (
                  <Message info>
                    Video Task (V1) uses async task creation and polling. Please enter supported model IDs manually in V1.
                  </Message>
                )}
                <div style={{ lineHeight: '40px', marginBottom: '12px' }}>
                  {!isVideoTaskChannelV1() && (
                    <>
                      <Button
                        type={'button'}
                        loading={discoveryLoading}
                        disabled={discoveryLoading || !canDiscoverModels()}
                        onClick={openModelPicker}
                      >
                        {t('channel.edit.buttons.fill_models')}
                      </Button>
                      <Button
                        type={'button'}
                        loading={discoveryLoading}
                        disabled={discoveryLoading || !canDiscoverModels()}
                        onClick={importAllModels}
                      >
                        {t('channel.edit.buttons.fill_all')}
                      </Button>
                      <Button
                        type={'button'}
                        loading={discoveryLoading}
                        disabled={discoveryLoading || !canDiscoverModels()}
                        onClick={clearDelistedModels}
                      >
                        {t('channel.edit.buttons.clear_delisted')}
                      </Button>
                    </>
                  )}
                  <Button
                    type={'button'}
                    onClick={() => {
                      handleInputChange(null, { name: 'models', value: [] });
                    }}
                  >
                    {t('channel.edit.buttons.clear')}
                  </Button>
                  <Input
                    action={
                      <Button type={'button'} onClick={addCustomModel}>
                        {t('channel.edit.buttons.add_custom')}
                      </Button>
                    }
                    placeholder={t('channel.edit.buttons.custom_placeholder')}
                    value={customModel}
                    onChange={(e, { value }) => {
                      setCustomModel(value);
                    }}
                    onKeyDown={(e) => {
                      if (e.key === 'Enter') {
                        addCustomModel();
                        e.preventDefault();
                      }
                    }}
                  />
                </div>
                {!isVideoTaskChannelV1() && !canDiscoverModels() && (
                  <Message info>{t('channel.edit.messages.base_url_required')}</Message>
                )}
              </>
            )}
            {inputs.type !== 43 && (
              <>
                <Form.Field>
                  <Form.TextArea
                    label={t('channel.edit.model_mapping')}
                    placeholder={`${t(
                      'channel.edit.model_mapping_placeholder'
                    )}\n${JSON.stringify(MODEL_MAPPING_EXAMPLE, null, 2)}`}
                    name='model_mapping'
                    onChange={handleInputChange}
                    value={inputs.model_mapping}
                    style={{
                      minHeight: 150,
                      fontFamily: 'JetBrains Mono, Consolas',
                    }}
                    autoComplete='new-password'
                  />
                </Form.Field>
                <Form.Field>
                  <Form.TextArea
                    label={t('channel.edit.system_prompt')}
                    placeholder={t('channel.edit.system_prompt_placeholder')}
                    name='system_prompt'
                    onChange={handleInputChange}
                    value={inputs.system_prompt}
                    style={{
                      minHeight: 150,
                      fontFamily: 'JetBrains Mono, Consolas',
                    }}
                    autoComplete='new-password'
                  />
                </Form.Field>
              </>
            )}
            {inputs.type === 33 && (
              <Form.Field>
                <Form.Input
                  label='Region'
                  name='region'
                  required
                  placeholder={t('channel.edit.aws_region_placeholder')}
                  onChange={handleConfigChange}
                  value={config.region}
                  autoComplete=''
                />
                <Form.Input
                  label='AK'
                  name='ak'
                  required
                  placeholder={t('channel.edit.aws_ak_placeholder')}
                  onChange={handleConfigChange}
                  value={config.ak}
                  autoComplete=''
                />
                <Form.Input
                  label='SK'
                  name='sk'
                  required
                  placeholder={t('channel.edit.aws_sk_placeholder')}
                  onChange={handleConfigChange}
                  value={config.sk}
                  autoComplete=''
                />
              </Form.Field>
            )}
            {inputs.type === 42 && (
              <Form.Field>
                <Form.Input
                  label='Region'
                  name='region'
                  required
                  placeholder={t('channel.edit.vertex_region_placeholder')}
                  onChange={handleConfigChange}
                  value={config.region}
                  autoComplete=''
                />
                <Form.Input
                  label={t('channel.edit.vertex_project_id')}
                  name='vertex_ai_project_id'
                  required
                  placeholder={t('channel.edit.vertex_project_id_placeholder')}
                  onChange={handleConfigChange}
                  value={config.vertex_ai_project_id}
                  autoComplete=''
                />
                <Form.Input
                  label={t('channel.edit.vertex_credentials')}
                  name='vertex_ai_adc'
                  required
                  placeholder={t('channel.edit.vertex_credentials_placeholder')}
                  onChange={handleConfigChange}
                  value={config.vertex_ai_adc}
                  autoComplete=''
                />
              </Form.Field>
            )}
            {inputs.type === 34 && (
              <Form.Input
                label={t('channel.edit.user_id')}
                name='user_id'
                required
                placeholder={t('channel.edit.user_id_placeholder')}
                onChange={handleConfigChange}
                value={config.user_id}
                autoComplete=''
              />
            )}
            {inputs.type !== 33 && inputs.type !== 42 && (
              <>
                <Header as='h4'>{t('channel.edit.key')}</Header>
                <Message info>{t('channel.edit.key_helper')}</Message>
                {keyRows.map((row, index) => {
                  const parsedKeyId = Number.parseInt(String(row.id), 10);
                  const canReenableKey =
                    row.status === 'disabled' && Number.isInteger(parsedKeyId);
                  const isRecoveringKey = recoveringKeyId === parsedKeyId;

                  return (
                    <div key={row.id} style={{ marginBottom: '12px' }}>
                      <Input
                      fluid
                      value={row.key_value}
                      placeholder={type2secretPrompt(inputs.type, t)}
                      onChange={(e, { value }) =>
                        updateKeyRow(index, { key_value: value })
                      }
                      action
                      >
                        <input />
                        <Button
                          type='button'
                          icon
                          onClick={() => moveKeyRow(index, -1)}
                          disabled={index === 0}
                          title={t('channel.edit.key_actions.up')}
                        >
                          <Icon name='arrow up' />
                        </Button>
                        <Button
                          type='button'
                          icon
                          onClick={() => moveKeyRow(index, 1)}
                          disabled={index === keyRows.length - 1}
                          title={t('channel.edit.key_actions.down')}
                        >
                          <Icon name='arrow down' />
                        </Button>
                        <Button
                          type='button'
                          icon
                          onClick={() => removeKeyRow(index)}
                          title={t('channel.edit.key_actions.remove')}
                        >
                          <Icon name='trash alternate outline' />
                        </Button>
                      </Input>
                      <div
                        style={{
                          marginTop: '8px',
                          display: 'flex',
                          gap: '8px',
                          flexWrap: 'wrap',
                        }}
                      >
                        <Label
                          as={canReenableKey ? 'a' : undefined}
                          color={getKeyStatusColor(row.status)}
                          onClick={canReenableKey ? () => handleEnableKeyV2(row) : undefined}
                          style={canReenableKey ? { cursor: 'pointer', opacity: isRecoveringKey ? 0.7 : 1 } : undefined}
                          title={canReenableKey ? t('channel.edit.key_actions.reenable') : undefined}
                        >
                          {isRecoveringKey
                            ? t('channel.edit.key_actions.reenable_loading')
                            : getKeyStatusText(row.status)}
                        </Label>
                        {row.last_used_at && (
                          <Label basic>
                            {t('channel.edit.key_meta.last_used')}:
                            {' '}
                            {formatDateTime(row.last_used_at)}
                          </Label>
                        )}
                        {row.cooldown_until && (
                          <Label basic color='orange'>
                            {t('channel.edit.key_meta.cooldown_until')}:
                            {' '}
                            {formatDateTime(row.cooldown_until)}
                          </Label>
                        )}
                      </div>
                      {row.last_error && (
                        <Message warning size='small'>
                          <strong>{t('channel.edit.key_meta.last_error')}:</strong>
                          {' '}
                          {row.last_error}
                        </Message>
                      )}
                    </div>
                  );
                })}
                <Button type='button' onClick={addKeyRow}>
                  <Icon name='plus' />
                  {t('channel.edit.key_actions.add')}
                </Button>
              </>
            )}
            {inputs.type === 37 && (
              <Form.Field>
                <Form.Input
                  label='Account ID'
                  name='user_id'
                  required
                  placeholder={
                    '请输入 Account ID，例如：d8d7c61dbc334c32d3ced580e4bf42b4'
                  }
                  onChange={handleConfigChange}
                  value={config.user_id}
                  autoComplete=''
                />
              </Form.Field>
            )}
            {inputs.type !== 3 &&
              inputs.type !== 33 &&
              inputs.type !== 8 &&
              inputs.type !== 50 &&
              inputs.type !== 52 &&
              inputs.type !== 22 && (
                <Form.Field>
                  <Form.Input
                      label={t('channel.edit.proxy_url')}
                    name='base_url'
                      placeholder={t('channel.edit.proxy_url_placeholder')}
                    onChange={handleInputChange}
                    value={inputs.base_url}
                    autoComplete='new-password'
                  />
                </Form.Field>
              )}
            {inputs.type === 22 && (
              <Form.Field>
                <Form.Input
                  label='私有部署地址'
                  name='base_url'
                  placeholder={
                    '请输入私有部署地址，格式为：https://fastgpt.run/api/openapi'
                  }
                  onChange={handleInputChange}
                  value={inputs.base_url}
                  autoComplete='new-password'
                />
              </Form.Field>
            )}
            <Button onClick={handleCancel}>
              {t('channel.edit.buttons.cancel')}
            </Button>
            <Button
              type={isEdit ? 'button' : 'submit'}
              positive
              onClick={submit}
            >
              {t('channel.edit.buttons.submit')}
            </Button>
            </Form>
          </Card.Content>
        </Card>
        <Modal
          open={modelPickerOpen}
          onClose={() => setModelPickerOpen(false)}
          size='small'
        >
          <Modal.Header>{t('channel.edit.buttons.fill_models')}</Modal.Header>
          <Modal.Content>
            <Header as='h5'>
              {discoveryResult
                ? getDiscoverySourceText(discoveryResult.source)
                : t('channel.edit.discovery.sources.dynamic')}
            </Header>
            <Input
              fluid
              icon='search'
              placeholder={t('channel.edit.discovery.search_placeholder')}
              value={discoverySearch}
              onChange={(e, { value }) => setDiscoverySearch(value)}
              style={{ marginBottom: '12px' }}
            />
            <div style={{ maxHeight: '360px', overflowY: 'auto' }}>
              {(discoveryResult?.models || [])
                .map((modelName) =>
                  applyModelIdPrefix(modelName, inputs.model_id_prefix)
                )
                .filter((modelName) =>
                  modelName.toLowerCase().includes(discoverySearch.toLowerCase())
                )
                .map((modelName) => (
                  <div key={modelName} style={{ marginBottom: '8px' }}>
                    <Checkbox
                      label={modelName}
                      checked={selectedDiscoveredModels.includes(modelName)}
                      onChange={() => toggleDiscoveredModel(modelName)}
                    />
                  </div>
                ))}
              {discoveredModelDisplayList.filter((modelName) =>
                modelName.toLowerCase().includes(discoverySearch.toLowerCase())
              ).length === 0 && (
                <Message>{t('channel.edit.discovery.empty')}</Message>
              )}
            </div>
          </Modal.Content>
          <Modal.Actions>
            <Button onClick={() => setModelPickerOpen(false)}>
              {t('channel.edit.buttons.cancel')}
            </Button>
            <Button positive onClick={confirmSelectedModels}>
              {t('channel.edit.discovery.confirm')}
            </Button>
          </Modal.Actions>
        </Modal>
      </div>
    );
  };

export default EditChannel;

