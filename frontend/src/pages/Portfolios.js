import React, { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import api from '../utils/api';
import {
  useApiState,
  FormModal,
  FormField,
  ActionButtons,
  ActionButton,
  LoadingSpinner,
  ErrorMessage,
} from '../components/shared';
import './Portfolios.css';

/**
 * Portfolio management page
 *
 * Displays all portfolios in a grid layout with CRUD operations. Each portfolio
 * card shows name, description, and action buttons for viewing, editing, deleting,
 * and archiving. The "exclude_from_overview" flag controls whether a portfolio
 * appears on the overview dashboard.
 *
 * @returns {JSX.Element} The portfolios management page
 */
const Portfolios = () => {
  const { data: portfolios, loading, error, execute: fetchPortfolios } = useApiState([]);
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [editingPortfolio, setEditingPortfolio] = useState(null);
  const [newPortfolio, setNewPortfolio] = useState({
    name: '',
    description: '',
    exclude_from_overview: false,
  });
  const [submitting, setSubmitting] = useState(false);
  const navigate = useNavigate();

  useEffect(() => {
    fetchPortfolios(() => api.get('/portfolio'));
  }, [fetchPortfolios]);

  const handleSubmit = async (e) => {
    e.preventDefault();
    setSubmitting(true);

    try {
      if (editingPortfolio) {
        await fetchPortfolios(
          () => api.put(`/portfolio/${editingPortfolio.id}`, editingPortfolio),
          {
            onSuccess: () => {
              setIsModalOpen(false);
              setEditingPortfolio(null);
            },
          }
        );
      } else {
        await fetchPortfolios(() => api.post('/portfolio', newPortfolio), {
          onSuccess: () => {
            setIsModalOpen(false);
            setNewPortfolio({ name: '', description: '', exclude_from_overview: false });
          },
        });
      }
    } catch (error) {
      console.error('Error saving portfolio:', error);
    } finally {
      setSubmitting(false);
    }
  };

  const handleDelete = async (id) => {
    if (window.confirm('Are you sure you want to delete this portfolio?')) {
      try {
        await api.delete(`/portfolio/${id}`);
        fetchPortfolios(() => api.get('/portfolio'));
      } catch (error) {
        console.error('Error deleting portfolio:', error);
      }
    }
  };

  const handleViewPortfolio = (portfolioId) => {
    navigate(`/portfolio/${portfolioId}`);
  };

  const handleArchive = async (portfolioId) => {
    try {
      await api.post(`/portfolio/${portfolioId}/archive`);
      fetchPortfolios(() => api.get('/portfolio'));
    } catch (error) {
      console.error('Error archiving portfolio:', error);
    }
  };

  const handleUnarchive = async (portfolioId) => {
    try {
      await api.post(`/portfolio/${portfolioId}/unarchive`);
      fetchPortfolios(() => api.get('/portfolio'));
    } catch (error) {
      console.error('Error unarchiving portfolio:', error);
    }
  };

  if (loading) return <LoadingSpinner message="Loading portfolios..." />;
  if (error)
    return (
      <ErrorMessage error={error} onRetry={() => fetchPortfolios(() => api.get('/portfolio'))} />
    );

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
            <ActionButtons className="portfolio-actions">
              <ActionButton variant="primary" onClick={() => handleViewPortfolio(portfolio.id)}>
                View Details
              </ActionButton>
              <ActionButton
                variant="secondary"
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
              </ActionButton>
              <ActionButton variant="danger" onClick={() => handleDelete(portfolio.id)}>
                Delete
              </ActionButton>
              {portfolio.isArchived ? (
                <ActionButton variant="info" onClick={() => handleUnarchive(portfolio.id)}>
                  Unarchive
                </ActionButton>
              ) : (
                <ActionButton variant="warning" onClick={() => handleArchive(portfolio.id)}>
                  Archive
                </ActionButton>
              )}
            </ActionButtons>
          </div>
        ))}
      </div>

      <FormModal
        isOpen={isModalOpen}
        onClose={() => {
          setIsModalOpen(false);
          setEditingPortfolio(null);
        }}
        title={editingPortfolio ? 'Edit Portfolio' : 'Add Portfolio'}
        onSubmit={handleSubmit}
        loading={submitting}
        submitText={editingPortfolio ? 'Update' : 'Create'}
        submitVariant={editingPortfolio ? 'primary' : 'success'}
      >
        <FormField
          label="Name"
          type="text"
          value={editingPortfolio ? editingPortfolio.name : newPortfolio.name}
          onChange={(value) => {
            if (editingPortfolio) {
              setEditingPortfolio({ ...editingPortfolio, name: value });
            } else {
              setNewPortfolio({ ...newPortfolio, name: value });
            }
          }}
          required
        />

        <FormField
          label="Description"
          type="textarea"
          value={editingPortfolio ? editingPortfolio.description : newPortfolio.description}
          onChange={(value) => {
            if (editingPortfolio) {
              setEditingPortfolio({ ...editingPortfolio, description: value });
            } else {
              setNewPortfolio({ ...newPortfolio, description: value });
            }
          }}
        />

        <FormField
          label="Exclude from overview page"
          type="checkbox"
          value={
            editingPortfolio
              ? editingPortfolio.exclude_from_overview
              : newPortfolio.exclude_from_overview
          }
          onChange={(value) => {
            if (editingPortfolio) {
              setEditingPortfolio({
                ...editingPortfolio,
                exclude_from_overview: value,
              });
            } else {
              setNewPortfolio({
                ...newPortfolio,
                exclude_from_overview: value,
              });
            }
          }}
        />
      </FormModal>
    </div>
  );
};

export default Portfolios;
