import React, { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import api from '../utils/api';
import Modal from '../components/Modal';
import './Portfolios.css';

const Portfolios = () => {
  const [portfolios, setPortfolios] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [editingPortfolio, setEditingPortfolio] = useState(null);
  const [newPortfolio, setNewPortfolio] = useState({
    name: '',
    description: '',
    exclude_from_overview: false,
  });
  const navigate = useNavigate();

  useEffect(() => {
    fetchPortfolios();
  }, []);

  const fetchPortfolios = async () => {
    try {
      const response = await api.get('/portfolios');
      setPortfolios(response.data);
      setError(null);
    } catch (err) {
      setError('Error fetching portfolios');
      console.error('Error:', err);
    } finally {
      setLoading(false);
    }
  };

  const handleSubmit = async (e) => {
    e.preventDefault();
    try {
      if (editingPortfolio) {
        const response = await api.put(`/portfolios/${editingPortfolio.id}`, editingPortfolio);
        setPortfolios(portfolios.map((p) => (p.id === editingPortfolio.id ? response.data : p)));
      } else {
        const response = await api.post('/portfolios', newPortfolio);
        setPortfolios([...portfolios, response.data]);
      }
      setIsModalOpen(false);
      setEditingPortfolio(null);
      setNewPortfolio({ name: '', description: '', exclude_from_overview: false });
    } catch (error) {
      console.error('Error saving portfolio:', error);
    }
  };

  const handleDelete = async (id) => {
    if (window.confirm('Are you sure you want to delete this portfolio?')) {
      try {
        await api.delete(`/portfolios/${id}`);
        setPortfolios(portfolios.filter((p) => p.id !== id));
      } catch (error) {
        console.error('Error deleting portfolio:', error);
      }
    }
  };

  const handleViewPortfolio = (portfolioId) => {
    navigate(`/portfolios/${portfolioId}`);
  };

  const handleArchive = async (portfolioId) => {
    try {
      await api.post(`/portfolios/${portfolioId}/archive`);
      fetchPortfolios();
    } catch (error) {
      console.error('Error archiving portfolio:', error);
    }
  };

  const handleUnarchive = async (portfolioId) => {
    try {
      await api.post(`/portfolios/${portfolioId}/unarchive`);
      fetchPortfolios();
    } catch (error) {
      console.error('Error unarchiving portfolio:', error);
    }
  };

  if (loading) return <div>Loading...</div>;
  if (error) return <div>Error: {error}</div>;

  return (
    <div className="portfolios-page">
      <div className="page-header">
        <h1>Portfolios</h1>
        <button onClick={() => setIsModalOpen(true)}>Add Portfolio</button>
      </div>

      <div className="portfolios-grid">
        {portfolios.map((portfolio) => (
          <div key={portfolio.id} className="portfolio-card">
            <h2>{portfolio.name}</h2>
            <p>{portfolio.description}</p>
            <div className="portfolio-actions">
              <button onClick={() => handleViewPortfolio(portfolio.id)}>View Details</button>
              <button
                onClick={() => {
                  setEditingPortfolio({
                    id: portfolio.id,
                    name: portfolio.name,
                    description: portfolio.description,
                    exclude_from_overview: portfolio.exclude_from_overview || false,
                  });
                  setIsModalOpen(true);
                }}
              >
                Edit
              </button>
              <button onClick={() => handleDelete(portfolio.id)}>Delete</button>
              {portfolio.is_archived ? (
                <button onClick={() => handleUnarchive(portfolio.id)}>Unarchive</button>
              ) : (
                <button onClick={() => handleArchive(portfolio.id)}>Archive</button>
              )}
            </div>
          </div>
        ))}
      </div>

      <Modal
        isOpen={isModalOpen}
        onClose={() => {
          setIsModalOpen(false);
          setEditingPortfolio(null);
        }}
        title={editingPortfolio ? 'Edit Portfolio' : 'Add Portfolio'}
      >
        <form onSubmit={handleSubmit}>
          <div className="form-group">
            <label>Name:</label>
            <input
              type="text"
              value={editingPortfolio ? editingPortfolio.name : newPortfolio.name}
              onChange={(e) => {
                if (editingPortfolio) {
                  setEditingPortfolio({ ...editingPortfolio, name: e.target.value });
                } else {
                  setNewPortfolio({ ...newPortfolio, name: e.target.value });
                }
              }}
              required
            />
          </div>
          <div className="form-group">
            <label>Description:</label>
            <textarea
              value={editingPortfolio ? editingPortfolio.description : newPortfolio.description}
              onChange={(e) => {
                if (editingPortfolio) {
                  setEditingPortfolio({ ...editingPortfolio, description: e.target.value });
                } else {
                  setNewPortfolio({ ...newPortfolio, description: e.target.value });
                }
              }}
            />
          </div>
          <div className="form-group">
            <label>
              <input
                type="checkbox"
                checked={
                  editingPortfolio
                    ? editingPortfolio.exclude_from_overview
                    : newPortfolio.exclude_from_overview
                }
                onChange={(e) => {
                  if (editingPortfolio) {
                    setEditingPortfolio({
                      ...editingPortfolio,
                      exclude_from_overview: e.target.checked,
                    });
                  } else {
                    setNewPortfolio({
                      ...newPortfolio,
                      exclude_from_overview: e.target.checked,
                    });
                  }
                }}
              />
              Exclude from overview page
            </label>
          </div>
          <div className="modal-actions">
            <button className={editingPortfolio ? 'edit-button' : 'add-button'} type="submit">{editingPortfolio ? 'Update' : 'Create'}</button>
            <button
              className="cancel-button"
              type="button"
              onClick={() => {
                setIsModalOpen(false);
                setEditingPortfolio(null);
              }}
            >
              Cancel
            </button>
          </div>
        </form>
      </Modal>
    </div>
  );
};

export default Portfolios;
