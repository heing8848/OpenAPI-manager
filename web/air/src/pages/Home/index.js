import React, { useContext, useEffect, useState } from 'react';
import { Card, Col, Row } from '@douyinfe/semi-ui';
import { API, showError, showNotice, timestamp2string } from '../../helpers';
import { StatusContext } from '../../context/Status';
import { marked } from 'marked';

const Home = () => {
  const [statusState] = useContext(StatusContext);
  const [homePageContentLoaded, setHomePageContentLoaded] = useState(false);
  const [homePageContent, setHomePageContent] = useState('');

  const displayNotice = async () => {
    const res = await API.get('/api/notice');
    const { success, message, data } = res.data;
    if (success) {
      const oldNotice = localStorage.getItem('notice');
      if (data !== oldNotice && data !== '') {
        const htmlNotice = marked(data);
        showNotice(htmlNotice, true);
        localStorage.setItem('notice', data);
      }
    } else {
      showError(message);
    }
  };

  const displayHomePageContent = async () => {
    setHomePageContent(localStorage.getItem('home_page_content') || '');
    const res = await API.get('/api/home_page_content');
    const { success, message, data } = res.data;
    if (success) {
      let content = data;
      if (!data.startsWith('https://')) {
        content = marked.parse(data);
      }
      setHomePageContent(content);
      localStorage.setItem('home_page_content', content);
    } else {
      showError(message);
      setHomePageContent('Failed to load homepage content...');
    }
    setHomePageContentLoaded(true);
  };

  const getStartTimeString = () => {
    const timestamp = statusState?.status?.start_time;
    return statusState?.status ? timestamp2string(timestamp) : '';
  };

  useEffect(() => {
    displayNotice().then();
    displayHomePageContent().then();
  }, []);

  return (
    <>
      {homePageContentLoaded && homePageContent === '' ? (
        <Card
          bordered={false}
          headerLine={false}
          title='System Status'
          bodyStyle={{ padding: '10px 20px' }}
        >
          <Row gutter={16}>
            <Col span={12}>
              <Card title='System Information'>
                <p>Name: {statusState?.status?.system_name}</p>
                <p>Version: {statusState?.status?.version || 'unknown'}</p>
                <p>Start Time: {getStartTimeString()}</p>
              </Card>
            </Col>
            <Col span={12}>
              <Card title='System Configuration'>
                <p>
                  Email Verification:{' '}
                  {statusState?.status?.email_verification ? 'Enabled' : 'Disabled'}
                </p>
                <p>
                  GitHub OAuth:{' '}
                  {statusState?.status?.github_oauth ? 'Enabled' : 'Disabled'}
                </p>
                <p>
                  WeChat Login:{' '}
                  {statusState?.status?.wechat_login ? 'Enabled' : 'Disabled'}
                </p>
                <p>
                  Turnstile Check:{' '}
                  {statusState?.status?.turnstile_check ? 'Enabled' : 'Disabled'}
                </p>
              </Card>
            </Col>
          </Row>
        </Card>
      ) : (
        <>
          {homePageContent.startsWith('https://') ? (
            <iframe
              src={homePageContent}
              style={{ width: '100%', height: '100vh', border: 'none' }}
            />
          ) : (
            <div
              style={{ fontSize: 'larger' }}
              dangerouslySetInnerHTML={{ __html: homePageContent }}
            ></div>
          )}
        </>
      )}
    </>
  );
};

export default Home;
