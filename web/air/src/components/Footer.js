import React, { useEffect, useState } from 'react';
import { Container, Segment } from 'semantic-ui-react';
import { getFooterHTML } from '../helpers';

const Footer = () => {
  const [footer, setFooter] = useState(getFooterHTML());
  let remainCheckTimes = 5;

  const loadFooter = () => {
    const footerHTML = localStorage.getItem('footer_html');
    if (footerHTML) {
      setFooter(footerHTML);
    }
  };

  useEffect(() => {
    const timer = setInterval(() => {
      if (remainCheckTimes <= 0) {
        clearInterval(timer);
        return;
      }
      remainCheckTimes--;
      loadFooter();
    }, 200);
    return () => clearTimeout(timer);
  }, []);

  return (
    <Segment vertical>
      <Container textAlign='center'>
        {footer ? (
          <div
            className='custom-footer'
            dangerouslySetInnerHTML={{ __html: footer }}
          ></div>
        ) : (
          <div className='custom-footer'>
            Alfred API项目参考GitHub开源项目One API (One API ) 由 Alfred改写功能.
          </div>
        )}
      </Container>
    </Segment>
  );
};

export default Footer;
