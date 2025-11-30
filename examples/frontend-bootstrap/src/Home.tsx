import { useState } from "react";
import { IndexRouteProps } from "./generated";
import { Container, Row, Col, Button, Card } from "react-bootstrap";
import GoLogo from "../public/go.png";
import ReactLogo from "../public/react.png";
import "bootstrap/dist/css/bootstrap.min.css";

function Home({ initialCount }: IndexRouteProps) {
  const [count, setCount] = useState(initialCount);

  return (
    <Container className="py-5">
      <Row className="justify-content-center text-center mb-4">
        <Col md={8}>
          <div className="d-flex justify-content-center align-items-center gap-3 mb-4">
            <img src={GoLogo} alt="Go logo" height={70} />
            <span className="fs-1">+</span>
            <img src={ReactLogo} alt="React logo" height={80} />
          </div>
          <h1 className="display-4 mb-3">Go + React + Bootstrap 5</h1>
          <p className="lead text-muted">
            Server-side rendered React with Go backend and Bootstrap 5 styling
          </p>
        </Col>
      </Row>

      <Row className="justify-content-center">
        <Col md={6}>
          <Card className="shadow-sm">
            <Card.Body className="text-center py-4">
              <Card.Title className="mb-4">Counter Example</Card.Title>
              <div className="d-flex justify-content-center align-items-center gap-3">
                <Button
                  variant="outline-primary"
                  size="lg"
                  onClick={() => setCount(count - 1)}
                >
                  -
                </Button>
                <span className="display-5 mx-3">{count}</span>
                <Button
                  variant="primary"
                  size="lg"
                  onClick={() => setCount(count + 1)}
                >
                  +
                </Button>
              </div>
            </Card.Body>
          </Card>
        </Col>
      </Row>

      <Row className="justify-content-center mt-4">
        <Col md={6} className="text-center">
          <a
            href="https://github.com/yejune/gotossr"
            target="_blank"
            className="btn btn-link"
          >
            View project on GitHub
          </a>
        </Col>
      </Row>
    </Container>
  );
}

export default Home;
